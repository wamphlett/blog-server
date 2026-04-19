package metrics

import (
	"context"
	"time"

	"log/slog"

	"github.com/getsentry/sentry-go"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/pkg/errors"

	"github.com/wamphlett/blog-server/config"
)

// Client defines a new metrics client
type Client struct {
	influx      influxdb2.Client
	writer      api.WriteAPIBlocking
	defaultTags map[string]string
}

// Options defines the function required for settings options
type Option func(*Client)

// New creates a new metrics client with the given options
func New(cfg *config.InfluxConfig, options ...Option) *Client {
	slog.Info("initialising new influxdb client")
	client := influxdb2.NewClient(cfg.Host, cfg.Token)
	c := &Client{
		influx:      client,
		writer:      client.WriteAPIBlocking(cfg.Org, cfg.Bucket),
		defaultTags: map[string]string{},
	}

	// apply options
	for _, option := range options {
		option(c)
	}

	return c
}

// WithDefaultTags specifies a list of tags to use on every metric
func WithDefaultTags(tags map[string]string) Option {
	return func(c *Client) {
		for tag, value := range tags {
			c.defaultTags[tag] = value
		}
	}
}

func (c *Client) publish(measurement string, fields map[string]interface{}, tags map[string]string) {
	p := influxdb2.NewPoint(measurement, mergeTags(tags, c.defaultTags), fields, time.Now())
	if err := c.writer.WritePoint(context.Background(), p); err != nil {
		slog.Error("failed to publish metric", "measurement", measurement, "error", err)
		sentry.CaptureException(errors.Wrap(err, "failed to publish metrics to influxdb"))
	}
}

func (c *Client) publishBatch(points []*write.Point) {
	if err := c.writer.WritePoint(context.Background(), points...); err != nil {
		slog.Error("failed to publish batch metrics", "count", len(points), "error", err)
		sentry.CaptureException(errors.Wrap(err, "failed to publish batch metrics to influxdb"))
	}
}

func mergeTags(base, tags map[string]string) map[string]string {
	for tag, value := range tags {
		base[tag] = value
	}
	return base
}

func noTags() map[string]string {
	return map[string]string{}
}
