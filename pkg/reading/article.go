package reading

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wamphlett/blog-server/pkg/model"
)

// loadArticleFromPath reads the given file path and creates a new article
func (r *Reader) LoadArticleFromFile(articleFilePath, topicSlug string) *model.Article {
	article := &model.Article{
		FilePath:  articleFilePath,
		TopicSlug: topicSlug,
		Metadata:  map[string]string{},
	}

	headers := r.parseFileHeaders(articleFilePath)

	for header, value := range headers {
		switch header {
		case "published":
			article.PublishedAt = convertToTimestamp(value)
		case "updated":
			article.UpdatedAt = convertToTimestamp(value)
		case "hidden":
			article.Hidden = value == "true"
		case "slug":
			article.Slug = strings.ToLower(value)
		case "title":
			article.Title = value
		case "description":
			article.Description = value
		case "image":
			article.Image = value
		case "priority":
			article.Priority, _ = strconv.ParseInt(value, 10, 64)
		default:
			article.Metadata[header] = value
		}
	}

	filename := strings.TrimRight(filepath.Base(articleFilePath), ".md")
	if article.Slug == "" {
		article.Slug = strings.ToLower(filename)
	}

	if article.Title == "" {
		article.Title = filename
	}

	article.URI = filepath.Join("/", topicSlug, article.Slug)

	return article
}
