package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/pkg/errors"
	log "unknwon.dev/clog/v2"

	"github.com/wamphlett/blog-server/config"
	"github.com/wamphlett/blog-server/pkg/indexing"
	database "github.com/wamphlett/blog-server/pkg/memoryDatabase"
	memorydatabase "github.com/wamphlett/blog-server/pkg/memoryDatabase"
	"github.com/wamphlett/blog-server/pkg/metrics"
	"github.com/wamphlett/blog-server/pkg/model"
	"github.com/wamphlett/blog-server/pkg/reading"
	"github.com/wamphlett/blog-server/pkg/scheduler"
	"github.com/wamphlett/blog-server/pkg/serving"
	"github.com/wamphlett/blog-server/pkg/updating"
)

func init() {
	err := log.NewConsole()
	if err != nil {
		panic("unable to create new logger: " + err.Error())
	}
}

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	cfg := config.NewFromEnv()
	bugsnag.Configure(bugsnag.Configuration{APIKey: cfg.BugsnagApiKey})

	// create a new metrics client
	metricsClient := metrics.New(cfg.Influx, metrics.WithDefaultTags(map[string]string{
		"environment": cfg.Environment,
	}))

	// create a new in memory database
	database := memorydatabase.New()

	// create a new indexer
	indexer := indexing.NewIndex(database, metricsClient)

	// create a new reader
	reader := reading.New(indexer, cfg.StaticAssetsURL, cfg.ContentAssetDir, metricsClient)

	// create a new updater
	_, err := updating.New(
		cfg.ContentPath,
		cfg.TopicFile,
		reader,
		metricsClient,
		updating.WithRemoteRepository(cfg.ContentRepo),
		updating.WithRefreshInterval(time.Duration(cfg.ContentUpdateIntervalSeconds)*time.Second),
		// the indexer directly receives the topics and articles every time the content is updated
		updating.WithReceiver(updateReceiver(cfg.BlogSiteHost, cfg.BlogSiteSecret, database, indexer)),
	)
	if err != nil {
		err = errors.Wrap(err, "failed to create updater")
		bugsnag.Notify(err)
		log.Fatal(err.Error())
	}

	// schedule a reindex every 24 hours
	scheduler := scheduler.New(time.Date(0, 0, 0, 0, 1, 0, 0, time.Local), func() {
		indexer.Reindex()
		invalidateSiteCaches(cfg.BlogSiteHost, "/", cfg.BlogSiteSecret)
	})

	// create and run a new server
	server := serving.New(reader, indexer, cfg.ContentPath, cfg.ContentAssetDir, cfg.TopicFile, metricsClient,
		serving.WithPort(cfg.ServerPort), serving.WithAllowedOrigins(cfg.ServerAllowedOrigins))
	go server.ListenAndServe()

	// wait for shutdown signals
	<-signals
	server.Shutdown()
	scheduler.Shutdown()
}

func updateReceiver(blogSitehost, secret string, db *database.Database, index *indexing.Index) func([]*model.Topic, []*model.Article) {
	firstReceive := true
	return func(updatedTopics []*model.Topic, updatedArticles []*model.Article) {
		for _, topic := range updatedTopics {
			db.StoreTopic(topic)
		}

		for _, article := range updatedArticles {
			db.StoreArticle(article)
		}

		if len(updatedTopics) > 0 || len(updatedArticles) > 0 {
			log.Info("reindexing after storing %d topics and %d articles", len(updatedTopics), len(updatedArticles))
			index.Reindex()
		}

		if firstReceive {
			log.Info("first receive, not clearing site cache")
			firstReceive = false
			return
		}

		if len(updatedTopics) == 0 && len(updatedArticles) == 0 {
			return
		}

		if blogSitehost == "" || secret == "" {
			log.Warn("not clearing site cache as blog site host or secret is not set")
			return
		}

		for _, topic := range updatedTopics {
			if err := invalidateSiteCaches(blogSitehost, topic.URI, secret); err != nil {
				log.Error("failed to invalidate site cache for topic %s: %v", topic.URI, err)
			}
		}

		for _, article := range updatedArticles {
			if err := invalidateSiteCaches(blogSitehost, article.URI, secret); err != nil {
				log.Error("failed to invalidate site cache for article %s: %v", article.URI, err)
			}
		}
	}
}

func invalidateSiteCaches(host, path, secret string) error {
	log.Info(fmt.Sprintf("invalidating site cache for %s", path))
	url := fmt.Sprintf("%s/api/revalidate?path=%s&secret=%s", host, path, secret)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
