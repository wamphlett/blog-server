package metrics

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/wamphlett/blog-server/pkg/telemetry"
)

// Client holds all OTEL metric instruments for the blog server.
type Client struct {
	httpRequestDuration otelmetric.Float64Histogram

	parseFileDuration    otelmetric.Float64Histogram
	parseHeadersDuration otelmetric.Float64Histogram

	contentUpdateDuration otelmetric.Float64Histogram
	contentUpdatesCount   otelmetric.Int64Counter
	lastContentUpdateNano atomic.Int64 // unix nanoseconds, read by the async content age gauge

	indexDuration otelmetric.Float64Histogram
	topicsCount   otelmetric.Int64Gauge
	articlesCount otelmetric.Int64Gauge
}

// New creates a new metrics client and registers all OTEL instruments.
func New() *Client {
	meter := otel.Meter(telemetry.InstrumentationName)

	c := &Client{
		httpRequestDuration: must(meter.Float64Histogram(
			semconv.HTTPServerRequestDurationName,
			otelmetric.WithUnit(semconv.HTTPServerRequestDurationUnit),
			otelmetric.WithDescription(semconv.HTTPServerRequestDurationDescription),
		)),
		parseFileDuration: must(meter.Float64Histogram(
			"blog.server.parse.file.duration",
			otelmetric.WithUnit("s"),
			otelmetric.WithDescription("Duration of markdown file parse operations"),
		)),
		parseHeadersDuration: must(meter.Float64Histogram(
			"blog.server.parse.headers.duration",
			otelmetric.WithUnit("s"),
			otelmetric.WithDescription("Duration of content header parse operations"),
		)),
		contentUpdateDuration: must(meter.Float64Histogram(
			"blog.server.content.update.duration",
			otelmetric.WithUnit("s"),
			otelmetric.WithDescription("Duration of content update operations"),
		)),
		contentUpdatesCount: must(meter.Int64Counter(
			"blog.server.content.updates",
			otelmetric.WithUnit("{update}"),
			otelmetric.WithDescription("Total number of content update cycles"),
		)),
		indexDuration: must(meter.Float64Histogram(
			"blog.server.index.duration",
			otelmetric.WithUnit("s"),
			otelmetric.WithDescription("Duration of content indexing operations"),
		)),
		topicsCount: must(meter.Int64Gauge(
			"blog.server.topics",
			otelmetric.WithUnit("{topic}"),
			otelmetric.WithDescription("Number of topics currently indexed"),
		)),
		articlesCount: must(meter.Int64Gauge(
			"blog.server.articles",
			otelmetric.WithUnit("{article}"),
			otelmetric.WithDescription("Number of articles currently indexed"),
		)),
	}

	// Async gauge: reports seconds since last content update so Grafana can
	// alert on freshness directly without a time() subtraction.
	contentAgeGauge := must(meter.Float64ObservableGauge(
		"blog.server.content.age",
		otelmetric.WithUnit("s"),
		otelmetric.WithDescription("Seconds since the most recent content update"),
	))
	must(meter.RegisterCallback(
		func(_ context.Context, o otelmetric.Observer) error {
			if t := c.lastContentUpdateNano.Load(); t != 0 {
				o.ObserveFloat64(contentAgeGauge, time.Since(time.Unix(0, t)).Seconds())
			}
			return nil
		},
		contentAgeGauge,
	))

	return c
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
