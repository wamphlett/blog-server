package indexing

import (
	"path/filepath"
	"strings"
)

// Article defines the information held about an article
type Article struct {
	Slug     string
	URI      string
	FilePath string
	Title    string
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

	fileName := strings.TrimRight(filepath.Base(articleFilePath), ".md")
	slug, ok := headers["slug"]
	if !ok {
		slug = strings.ToLower(fileName)
	}
	title, ok := headers["title"]
	if !ok {
		title = fileName
	}
	return &Article{
		Title:    title,
		URI:      filepath.Join("/", topicSlug, slug),
		Slug:     strings.Trim(slug, "/"),
		FilePath: articleFilePath,
	}
}
