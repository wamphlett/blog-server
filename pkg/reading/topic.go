package reading

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wamphlett/blog-server/pkg/model"
)

// loadTopicFromFile creates a new topic from file at the given path
func (r *Reader) LoadTopicFromFile(topicFilePath string) *model.Topic {
	topic := &model.Topic{
		FilePath: topicFilePath,
		Metadata: map[string]string{},
	}

	headers := r.parseFileHeaders(topicFilePath)

	for header, value := range headers {
		switch header {
		case "published":
			topic.PublishedAt = convertToTimestamp(value)
		case "updated":
			topic.UpdatedAt = convertToTimestamp(value)
		case "hidden":
			topic.Hidden = value == "true"
		case "slug":
			topic.Slug = strings.ToLower(value)
		case "title":
			topic.Title = value
		case "description":
			topic.Description = value
		case "image":
			topic.Image = value
		case "priority":
			topic.Priority, _ = strconv.ParseInt(value, 10, 64)
		default:
			topic.Metadata[header] = value
		}
	}

	topicDirName := filepath.Base(filepath.Dir(topicFilePath))

	if topic.Slug == "" {
		topic.Slug = strings.ToLower(topicDirName)
	}

	if topic.Title == "" {
		topic.Title = topicDirName
	}

	topic.URI = filepath.Join("/", topic.Slug)

	return topic
}
