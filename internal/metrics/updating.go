package metrics

import "time"

func (c *Client) ContentUpdated(startTime time.Time) {
	fields := map[string]interface{}{
		"time_taken_ms":    time.Since(startTime).Milliseconds(),
		"update_time_unix": startTime.UnixMilli(),
		"count":            1,
	}
	c.publish("content_updated", fields, noTags())
}
