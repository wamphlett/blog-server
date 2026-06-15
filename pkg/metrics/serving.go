package metrics

import (
	"strconv"
	"time"
)

// Request records the duration and count of an HTTP request.
func (c *Client) Request(method, path string, status int, startTime time.Time) {
	statusCode := strconv.Itoa(status)
	duration := time.Since(startTime).Seconds()
	c.httpRequestDuration.WithLabelValues(method, path, statusCode).Observe(duration)
	c.httpRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
}
