package metrics

import "time"

// ContentUpdated records a completed content update cycle.
func (c *Client) ContentUpdated(startTime time.Time) {
	c.contentUpdateDuration.Observe(time.Since(startTime).Seconds())
	c.contentUpdatesTotal.Inc()
	c.contentLastUpdated.SetToCurrentTime()
}
