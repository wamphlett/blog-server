package indexing

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Article defines the information held about an article
type Article struct {
	Slug        string
	URI         string
	FilePath    string
	Title       string
	Description string
	Image       string
	Priority    int64
	Metadata    map[string]string
}

// GetURI returns the URL used by the website
func (a *Article) GetURI() string {
	return a.URI
}

// loadArticleFromPath reads the given file path and creates a new article
func (i *Index) loadArticleFromPath(articleFilePath, topicSlug string) *Article {
	headers := i.parseFileHeaders(articleFilePath)
	if published, ok := headers["published"]; !ok || published != "true" {
		return nil
	}

	article := &Article{
		FilePath: articleFilePath,
		Metadata: map[string]string{},
	}

	for header, value := range headers {
		switch header {
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
