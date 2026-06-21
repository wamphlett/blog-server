package metrics

import (
	"context"
	"time"
)

// ContentUpdated records a completed content update cycle.
func (c *Client) ContentUpdated(ctx context.Context, startTime time.Time) {
	c.contentUpdateDuration.Record(ctx, time.Since(startTime).Seconds())
	c.contentUpdatesCount.Add(ctx, 1)
	c.lastContentUpdateNano.Store(time.Now().UnixNano())
}
