package metrics

import (
	"context"
	"time"
)

// ParseFile records the duration of a markdown file parse operation.
func (c *Client) ParseFile(ctx context.Context, startTime time.Time) {
	c.parseFileDuration.Record(ctx, time.Since(startTime).Seconds())
}

// ParseHeaders records the duration of a content header parse operation.
func (c *Client) ParseHeaders(ctx context.Context, startTime time.Time) {
	c.parseHeadersDuration.Record(ctx, time.Since(startTime).Seconds())
}
