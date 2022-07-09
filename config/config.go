package config

import (
	"context"
	"log"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	// If specified, the updater will clone and fetch the content from the given remote git repository
	ContentRepo string `env:"CONTENT_REPO"`
	// The directory where the content is stored
	ContentPath     string `env:"CONTENT_PATH,default=./content"`
	ContentAssetDir string `env:"CONTENT_ASSET_DIR,default=images"`
	StaticAssetsURL string `env:"STATIC_ASSET_URL,required"`

	TopicFile string `env:"TOPIC_FILE,default=README.md"`
}

func NewFromEnv() *Config {
	ctx := context.Background()

	c := &Config{}
	if err := envconfig.Process(ctx, c); err != nil {
		log.Fatal(err)
	}

	return c
}
