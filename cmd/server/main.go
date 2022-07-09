package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/wamphlett/blog-server/config"
	"github.com/wamphlett/blog-server/internal/indexing"
	"github.com/wamphlett/blog-server/internal/reading"
	"github.com/wamphlett/blog-server/internal/serving"
	"github.com/wamphlett/blog-server/internal/updating"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	cfg := config.NewFromEnv()

	indexer := indexing.NewIndex(cfg.ContentPath, cfg.TopicFile)

	// start an updater if a repo was provided
	if cfg.ContentRepo != "" {
		onRefreshFunc := func() {
			if err := indexer.Index(); err != nil {
				log.Println(errors.Wrap(err, "failed to index"))
			}
		}
		if _, err := updating.New(cfg.ContentRepo, cfg.ContentPath, 20*time.Second, onRefreshFunc); err != nil {
			log.Fatal(errors.Wrap(err, "failed to create updater").Error())
		}
	}

	if err := indexer.Index(); err != nil {
		log.Println(errors.Wrap(err, "failed to index"))
	}

	reader := reading.New()

	server := serving.New(reader, indexer, cfg.ContentPath, cfg.ContentAssetDir, cfg.TopicFile)
	go server.ListenAndServe()

	<-signals
	server.Shutdown()
}
