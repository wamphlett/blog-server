package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"

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

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	cfg, err := config.NewFromEnv()
	setupLogger(cfg.LogLevel, cfg.LogFormat)
	if err := sentry.Init(sentry.ClientOptions{Dsn: cfg.SentryDSN}); err != nil {
		slog.Error("failed to initialise sentry", "error", err)
	}
	if err != nil {
		sentry.CaptureException(err)
		slog.Error("failed to load config", "error", err)
		sentry.Flush(2 * time.Second)
		os.Exit(1)
	}

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
	_, err = updating.New(
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
		sentry.CaptureException(err)
		slog.Error("failed to create updater", "error", err)
		os.Exit(1)
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
			slog.Info("reindexing after storing topics and articles", "topics", len(updatedTopics), "articles", len(updatedArticles))
			index.Reindex()
		}

		if firstReceive {
			slog.Info("first receive, not clearing site cache")
			firstReceive = false
			return
		}

		if len(updatedTopics) == 0 && len(updatedArticles) == 0 {
			return
		}

		if blogSitehost == "" || secret == "" {
			slog.Warn("not clearing site cache as blog site host or secret is not set")
			return
		}

		for _, topic := range updatedTopics {
			if err := invalidateSiteCaches(blogSitehost, topic.URI, secret); err != nil {
				slog.Error("failed to invalidate site cache for topic", "uri", topic.URI, "error", err)
			}
		}

		for _, article := range updatedArticles {
			if err := invalidateSiteCaches(blogSitehost, article.URI, secret); err != nil {
				slog.Error("failed to invalidate site cache for article", "uri", article.URI, "error", err)
			}
		}
	}
}

func setupLogger(level, format string) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: l}
	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func invalidateSiteCaches(host, path, secret string) error {
	slog.Info("invalidating site cache", "path", path)
	url := fmt.Sprintf("%s/api/revalidate?path=%s&secret=%s", host, path, secret)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
