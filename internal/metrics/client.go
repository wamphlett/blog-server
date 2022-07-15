package metrics

import (
	"context"
	"time"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/go-clog/clog"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/pkg/errors"

	"github.com/wamphlett/blog-server/config"
)

type Client struct {
	influx      influxdb2.Client
	writer      api.WriteAPIBlocking
	defaultTags map[string]string
}

type Option func(*Client)

func New(cfg *config.InfluxConfig, options ...Option) *Client {
	clog.Info("Initialising new influxdb client")
	client := influxdb2.NewClient(cfg.Host, cfg.Token)
	c := &Client{
		influx:      client,
		writer:      client.WriteAPIBlocking(cfg.Org, cfg.Bucket),
		defaultTags: map[string]string{},
	}

	for _, option := range options {
		option(c)
	}

	return c
}

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
		_ = bugsnag.Notify(errors.Wrap(err, "failed to publish metrics to influxdb"))
	}
}

func (c *Client) publishBatch(points []*write.Point) {
	if err := c.writer.WritePoint(context.Background(), points...); err != nil {
		_ = bugsnag.Notify(errors.Wrap(err, "failed to publish batch metrics to influxdb"))
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
