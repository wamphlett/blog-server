package indexing

import (
	"os"
	"path/filepath"
	"strings"
)

type Topic struct {
	Slug     string
	URI      string
	FilePath string
	Title    string
	Articles []*Article
}

func (i *Index) loadTopicFromFile(topicFilePath string) *Topic {
	headers := i.parseFileHeaders(topicFilePath)
	if published, ok := headers["published"]; !ok || published != "true" {
		return nil
	}

	topicDirName := filepath.Base(filepath.Dir(topicFilePath))
	slug, ok := headers["slug"]
	if !ok {
		slug = strings.ToLower(topicDirName)
	}

	title, ok := headers["title"]
	if !ok {
		title = topicDirName
	}

	topic := &Topic{
		Title:    title,
		URI:      filepath.Join("/", slug),
		Slug:     strings.Trim(slug, "/"),
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

		article := i.loadArticleFromPath(filepath.Join(filepath.Dir(topicFilePath), file.Name()), topic.Slug)
		if article != nil {
			topic.Articles = append(topic.Articles, article)
		}
	}

	if len(topic.Articles) == 0 {
		return nil
	}

	return topic
}

func (i *Index) GetTopic(slug string) *Topic {
	for _, topic := range i.topics {
		if topic.Slug == slug {
			return topic
		}
	}
	return nil
}

func (t *Topic) GetArticle(slug string) *Article {
	for _, article := range t.Articles {
		if article.Slug == slug {
			return article
		}
	}
	return nil
}

func (t *Topic) GetURI() string {
	return t.URI
}
