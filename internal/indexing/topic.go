package indexing

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Topic defines a topic entry
type Topic struct {
	Hidden      bool
	Slug        string
	URI         string
	FilePath    string
	Title       string
	Description string
	Image       string
	Metadata    map[string]string
	Priority    int64
	PublishedAt int64
	UpdatedAt   int64
	Articles    []*Article
}

// loadTopicFromFile creates a new topic from file at the given path
func (i *Index) loadTopicFromFile(topicFilePath string) *Topic {
	headers := i.parseFileHeaders(topicFilePath)

	topic := &Topic{
		FilePath: topicFilePath,
		Metadata: map[string]string{},
		Articles: []*Article{},
	}

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

// GetTopic returns the topic with the given slug
func (i *Index) GetTopic(slug string) *Topic {
	for _, topic := range i.topics {
		if topic.Slug == slug {
			return topic
		}
	}
	return nil
}

// GetArticle returns the article with the given slug
func (t *Topic) GetArticle(slug string) *Article {
	for _, article := range t.Articles {
		if article.Slug == slug {
			return article
		}
	}
	return nil
}

// GetURI returns the URI used by the website
func (t *Topic) GetURI() string {
	return t.URI
}
