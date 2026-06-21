package metrics

import (
	"context"
	"time"

	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Request records the duration of an HTTP request.
func (c *Client) Request(ctx context.Context, method string, status int, startTime time.Time) {
	c.httpRequestDuration.Record(ctx, time.Since(startTime).Seconds(),
		otelmetric.WithAttributes(
			semconv.HTTPRequestMethodKey.String(method),
			semconv.HTTPResponseStatusCodeKey.Int(status),
		),
	)
}
