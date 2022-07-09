package indexing

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type Metrics interface {
	Indexed()
}

type Index struct {
	contentPath string
	topicFile   string

	topics         []*Topic
	articlesByPath map[string]*Article
}

func NewIndex(contentPath, topicFile string) *Index {
	return &Index{
		topicFile:   topicFile,
		contentPath: contentPath,
	}
}

func (i *Index) Index() error {
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
	i.indexPaths()
	return nil
}

func (i *Index) GetTopics() []*Topic {
	return i.topics
}

func (i *Index) GetArticleByPath(path string) (*Article, error) {
	if i.articlesByPath == nil {
		i.indexPaths()
	}
	path = strings.TrimLeft(path, "/")
	if article, ok := i.articlesByPath[path]; ok {
		return article, nil
	}
	return nil, errors.New("no such article")
}

func (i *Index) indexPaths() {
	i.articlesByPath = make(map[string]*Article)
	for _, topic := range i.topics {
		for _, article := range topic.Articles {
			i.articlesByPath[strings.TrimLeft(filepath.Join(topic.Path, article.Path), "/")] = article
		}
	}
}
