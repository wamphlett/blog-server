package config

import (
	"context"
	"log"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
)

// Config defines the config values required to run the app
type Config struct {
	Environment          string   `env:"ENVIRONMENT,default=development"`
	ServerPort           int      `env:"PORT,default=3000"`
	ServerAllowedOrigins []string `env:"ALLOWED_ORIGINS"`
	// If specified, the updater will clone and fetch the content from the given remote git repository
	ContentRepo                  string `env:"CONTENT_REPO"`
	ContentUpdateIntervalSeconds int64  `env:"CONTENT_UPDATE_INTERVAL_SECONDS,default=300"`
	// The directory where the content is stored
	// This is where any remote repositories will be cloned to
	ContentPath string `env:"CONTENT_PATH,default=./content"`
	// The directory within the content path which holds static content
	ContentAssetDir string `env:"CONTENT_ASSET_DIR,default=images"`
	// The URL where static content will be served from
	StaticAssetsURL string `env:"STATIC_ASSET_URL,default=images"`

	// The host of the blog site
	BlogSiteHost   string `env:"BLOG_SITE_HOST"`
	BlogSiteSecret string `env:"BLOG_SITE_SECRET"`

	// The name which topic files use, everything else will be considered an article
	TopicFile string `env:"TOPIC_FILE,default=README.md"`

	Influx        *InfluxConfig
	BugsnagApiKey string `env:"BUGSNAG_API_KEY"`
}

// InfluxConfig defines the config to load InfluxDB
type InfluxConfig struct {
	Host   string `env:"INFLUX_HOST"`
	Bucket string `env:"INFLUX_BUCKET"`
	Token  string `env:"INFLUX_TOKEN"`
	Org    string `env:"INFLUX_ORG"`
}

// NewFromEnv reads the environment variables and creates a new config
func NewFromEnv() *Config {
	ctx := context.Background()

	c := &Config{}
	if err := envconfig.Process(ctx, c); err != nil {
		err = errors.Wrap(err, "failed to load config from env")
		bugsnag.Notify(err)
		log.Fatal(err)
	}

	return c
}
