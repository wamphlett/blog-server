package indexing

import (
	"os"
	"path/filepath"
	"strings"
)

type Topic struct {
	Path     string
	FilePath string
	Title    string
	Articles []*Article
}

func (i *Index) loadTopicFromFile(topicFilePath string) *Topic {
	headers := i.parseFileHeaders(topicFilePath)
	if published, ok := headers["published"]; !ok || published != "true" {
		return nil
	}

	path, ok := headers["path"]
	if !ok {
		path = filepath.Base(filepath.Dir(topicFilePath))
	}

	title, ok := headers["title"]
	if !ok {
		title = filepath.Base(filepath.Dir(topicFilePath))
	}

	topic := &Topic{
		Title:    title,
		Path:     strings.Trim(path, "/"),
		FilePath: topicFilePath,
		Articles: []*Article{},
	}

	files, err := os.ReadDir(filepath.Dir(topicFilePath))
	if err != nil {
		return nil
	}

	for _, file := range files {
		if file.IsDir() || file.Name() == filepath.Base(topicFilePath) || filepath.Ext(file.Name()) != ".md" {
			continue
		}

		article := i.loadArticleFromPath(filepath.Join(filepath.Dir(topicFilePath), file.Name()))
		if article != nil {
			topic.Articles = append(topic.Articles, article)
		}
	}

	if len(topic.Articles) == 0 {
		return nil
	}

	return topic
}

func (i *Index) GetTopic(path string) *Topic {
	for _, topic := range i.topics {
		if topic.Path == path {
			return topic
		}
	}
	return nil
}

func (t *Topic) GetArticle(path string) *Article {
	for _, article := range t.Articles {
		if article.Path == path {
			return article
		}
	}
	return nil
}
