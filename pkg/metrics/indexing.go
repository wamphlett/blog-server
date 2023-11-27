package metrics

import "time"

// Indexed records every time the content was indexed
func (c *Client) Indexed(startTime time.Time, topicCount, articleCount int) {
	fields := map[string]interface{}{
		"time_taken_ms": time.Since(startTime).Milliseconds(),
		"count":         1,
		"topic_count":   topicCount,
		"article_count": articleCount,
	}
	c.publish("indexed", fields, noTags())
}

// ParseHeaders records every time headers were parsed
func (c *Client) ParseHeaders(startTime time.Time) {
	fields := map[string]interface{}{
		"time_taken_ms": time.Since(startTime).Milliseconds(),
		"count":         1,
	}
	c.publish("parse_headers", fields, noTags())
}

// ReadFile records every time a file was read
func (c *Client) ReadFile(fileType string) {
	fields := map[string]interface{}{
		"file_type": fileType,
		"count":     1,
	}
	c.publish("read_file", fields, noTags())
}
