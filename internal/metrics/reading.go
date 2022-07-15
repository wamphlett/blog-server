package metrics

import "time"

func (c *Client) ParseFile(startTime time.Time) {
	fields := map[string]interface{}{
		"time_taken_ms": time.Since(startTime).Milliseconds(),
		"count":         1,
	}
	c.publish("parse_file", fields, noTags())
}
