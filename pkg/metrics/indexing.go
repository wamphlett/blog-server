package metrics

import "time"

// Indexed records a completed indexing operation.
func (c *Client) Indexed(startTime time.Time, topicCount, articleCount int) {
	c.indexDuration.Observe(time.Since(startTime).Seconds())
	c.indexTotal.Inc()
	c.topicsTotal.Set(float64(topicCount))
	c.articlesTotal.Set(float64(articleCount))
}

// ParseHeaders records the duration of a content header parse operation.
func (c *Client) ParseHeaders(startTime time.Time) {
	c.parseHeadersDuration.Observe(time.Since(startTime).Seconds())
	c.parseHeadersTotal.Inc()
}
