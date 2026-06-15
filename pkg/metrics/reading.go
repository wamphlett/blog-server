package metrics

import "time"

// ParseFile records the duration of a markdown file parse operation.
func (c *Client) ParseFile(startTime time.Time) {
	c.parseFileDuration.Observe(time.Since(startTime).Seconds())
	c.parseFileTotal.Inc()
}
