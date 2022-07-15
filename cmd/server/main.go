package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/pkg/errors"
	log "unknwon.dev/clog/v2"

	"github.com/wamphlett/blog-server/config"
	"github.com/wamphlett/blog-server/internal/indexing"
	"github.com/wamphlett/blog-server/internal/metrics"
	"github.com/wamphlett/blog-server/internal/reading"
	"github.com/wamphlett/blog-server/internal/serving"
	"github.com/wamphlett/blog-server/internal/updating"
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

	// create a new indexer
	indexer := indexing.NewIndex(cfg.ContentPath, cfg.TopicFile, metricsClient)

	// start an updater if a repo was provided
	if cfg.ContentRepo != "" {
		if _, err := updating.New(cfg.ContentRepo, cfg.ContentPath, time.Duration(cfg.ContentUpdateIntervalSeconds)*time.Second, metricsClient,
			updating.WithCallback(func() {
				indexContent(indexer)
			})); err != nil {
			err = errors.Wrap(err, "failed to create updater")
			bugsnag.Notify(err)
			log.Fatal(err.Error())
		}
	}

	indexContent(indexer)

	// create a new reader
	reader := reading.New(indexer, cfg.StaticAssetsURL, cfg.ContentAssetDir, metricsClient)

	// create and run a new server
	server := serving.New(reader, indexer, cfg.ContentPath, cfg.ContentAssetDir, cfg.TopicFile, metricsClient,
		serving.WithPort(cfg.ServerPort), serving.WithAllowedOrigins(cfg.ServerAllowedOrigins))
	go server.ListenAndServe()

	// wait for shutdown signals
	<-signals
	server.Shutdown()
}

func indexContent(index *indexing.Index) {
	if err := index.Index(); err != nil {
		err = errors.Wrap(err, "failed to index content")
		log.Error(err.Error())
		bugsnag.Notify(err)
	}
}
