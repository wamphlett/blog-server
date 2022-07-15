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

	metricsClient := metrics.New(cfg.Influx, metrics.WithDefaultTags(map[string]string{
		"environment": cfg.Environment,
	}))
	indexer := indexing.NewIndex(cfg.ContentPath, cfg.TopicFile, metricsClient)

	// start an updater if a repo was provided
	if cfg.ContentRepo != "" {
		onRefreshFunc := func() {
			if err := indexer.Index(); err != nil {
				log.Error("failed to index content", err)
			}
		}
		if _, err := updating.New(cfg.ContentRepo, cfg.ContentPath, time.Duration(cfg.ContentUpdateIntervalSeconds)*time.Second, metricsClient, onRefreshFunc); err != nil {
			log.Fatal(errors.Wrap(err, "failed to create updater").Error())
		}
	}

	if err := indexer.Index(); err != nil {
		log.Error("failed to index content", err)
	}

	reader := reading.New(indexer, cfg.StaticAssetsURL, cfg.ContentAssetDir, metricsClient)

	server := serving.New(reader, indexer, cfg.ContentPath, cfg.ContentAssetDir, cfg.TopicFile, metricsClient,
		serving.WithPort(cfg.ServerPort), serving.WithAllowedOrigins(cfg.ServerAllowedOrigins))
	go server.ListenAndServe()

	<-signals
	server.Shutdown()
}
