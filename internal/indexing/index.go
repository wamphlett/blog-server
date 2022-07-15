package indexing

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Metrics defines the metrics used by the index
type Metrics interface {
	Indexed(startTime time.Time, topicCount, articleCount int)
	ParseHeaders(startTime time.Time)
	ReadFile(fileType string)
}

// Indexable defines the methods required by an indexable item
type Indexable interface {
	GetURI() string
}

// Index defines an index
type Index struct {
	contentPath string
	topicFile   string

	topics          []*Topic
	articlesByURI   map[string]*Article
	entryByFilePath map[string]Indexable

	metrics Metrics
}

// NewIndex creates a new index with the required dependencies
func NewIndex(contentPath, topicFile string, metrics Metrics) *Index {
	return &Index{
		topicFile:   topicFile,
		contentPath: contentPath,
		metrics:     metrics,
	}
}

// Index will reindex the content directory
func (i *Index) Index() error {
	startTime := time.Now()
	// read the main content directory to look for topic directories
	files, err := os.ReadDir(i.contentPath)
	if err != nil {
		return errors.Wrap(err, "failed to read content directory")
	}

	topics := []*Topic{}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		topicFilePath := filepath.Join(i.contentPath, file.Name(), i.topicFile)
		if _, err := os.Stat(topicFilePath); os.IsNotExist(err) {
			continue
		}

		topic := i.loadTopicFromFile(topicFilePath)
		if topic == nil {
			continue
		}

		topics = append(topics, topic)
	}

	i.topics = topics
	i.indexURIs()
	i.indexFilePaths()

	articleCount := 0
	for _, topic := range topics {
		articleCount += len(topic.Articles)
	}
	i.metrics.Indexed(startTime, len(topics), articleCount)

	return nil
}

// GetTopics returns all the indexed topics
func (i *Index) GetTopics() []*Topic {
	return i.topics
}

// GetArticleByURI returns the article
func (i *Index) GetArticleByURI(uri string) (*Article, error) {
	if i.articlesByURI == nil {
		i.indexURIs()
	}
	uri = strings.TrimLeft(uri, "/")
	if article, ok := i.articlesByURI[uri]; ok {
		return article, nil
	}
	return nil, errors.New("no such article")
}

// GetURIForFile returns the URI used by the file at the given path
func (i *Index) GetURIForFile(filepath string) string {
	if entry, ok := i.entryByFilePath[filepath]; ok {
		return entry.GetURI()
	}
	return ""
}

// indexURIs stores entries by their URI
func (i *Index) indexURIs() {
	i.articlesByURI = make(map[string]*Article)
	for _, topic := range i.topics {
		for _, article := range topic.Articles {
			i.articlesByURI[strings.TrimLeft(filepath.Join(topic.Slug, article.Slug), "/")] = article
		}
	}
}

// indexFilePaths stores entries by their filepath on disk
func (i *Index) indexFilePaths() {
	i.entryByFilePath = make(map[string]Indexable)
	for _, topic := range i.topics {
		i.entryByFilePath[topic.FilePath] = topic
		for _, article := range topic.Articles {
			i.entryByFilePath[article.FilePath] = article
		}
	}
}
