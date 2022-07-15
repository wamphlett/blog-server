package indexing

import (
	"path/filepath"
	"strings"
)

type Article struct {
	Slug     string
	URI      string
	FilePath string
	Title    string
}

func (a *Article) GetURI() string {
	return a.URI
}

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

func (i *Index) GetArticle(topicSlug, articleSlug string) *Article {
	topic := i.GetTopic(topicSlug)
	if topic == nil {
		return nil
	}
	for _, article := range topic.Articles {
		if article.Slug == articleSlug {
			return article
		}
	}
	return nil
}
