package metrics

import "time"

// Request records how many requests the server has handled
func (c *Client) Request(uri string, startTime time.Time) {
	fields := map[string]interface{}{
		"time_taken_ms": time.Since(startTime).Milliseconds(),
		"count":         1,
	}
	tags := map[string]string{
		"uri": uri,
	}
	c.publish("request", fields, tags)
}
