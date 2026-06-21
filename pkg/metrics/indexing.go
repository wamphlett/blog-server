package metrics

import (
	"context"
	"time"
)

// Indexed records a completed indexing operation.
func (c *Client) Indexed(ctx context.Context, startTime time.Time, topicCount, articleCount int) {
	c.indexDuration.Record(ctx, time.Since(startTime).Seconds())
	c.topicsCount.Record(ctx, int64(topicCount))
	c.articlesCount.Record(ctx, int64(articleCount))
}
