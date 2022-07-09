package indexing

import (
	"path/filepath"
	"strings"
)

type Article struct {
	Path     string
	FilePath string
	Title    string
}

func (i *Index) loadArticleFromPath(articleFilePath string) *Article {
	headers := i.parseFileHeaders(articleFilePath)
	if published, ok := headers["published"]; !ok || published != "true" {
		return nil
	}
	path, ok := headers["path"]
	if !ok {
		path = filepath.Base(articleFilePath)
	}
	title, ok := headers["title"]
	if !ok {
		title = strings.TrimRight(filepath.Base(articleFilePath), ".md")
	}
	return &Article{
		Title:    title,
		Path:     path,
		FilePath: articleFilePath,
	}
}

func (i *Index) GetArticle(topicPath, articlePath string) *Article {
	topic := i.GetTopic(topicPath)
	if topic == nil {
		return nil
	}
	for _, article := range topic.Articles {
		if article.Path == articlePath {
			return article
		}
	}
	return nil
}
