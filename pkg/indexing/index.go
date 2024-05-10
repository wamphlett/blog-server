package indexing

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wamphlett/blog-server/pkg/model"
)

type Database interface {
	GetAllTopics() []*model.Topic
	GetAllArticles() []*model.Article
}

type ReindexedCallback func(result ReindexResults)

type Option func(*Index)

func WithReindexedCallback(callback ReindexedCallback) Option {
	return func(i *Index) {
		i.reindexedCallbacks = append(i.reindexedCallbacks, callback)
	}
}

// Metrics defines the metrics used by the index
type Metrics interface {
	Indexed(startTime time.Time, topicCount, articleCount int)
}

type ReindexResults struct {
	NewTopics       []*model.Topic
	UpdatedTopics   []*model.Topic
	NewArticles     []*model.Article
	UpdatedArticles []*model.Article
}

// Index defines an index
type Index struct {
	// where the content is stored
	reindexedCallbacks []ReindexedCallback

	// indexes
	topicsByIdentifier   map[string]*model.Topic
	articlesByIdentifier map[string]map[string]*model.Article
	articlesByTime       []*model.Article
	articlesByURI        map[string]*model.Article
	urisByFilepath       map[string]string

	// last indexed time
	lastIndexed time.Time

	database Database
	metrics  Metrics
}

// NewIndex creates a new index with the required dependencies
func NewIndex(database Database, metrics Metrics, opts ...Option) *Index {
	i := &Index{
		database: database,
		metrics:  metrics,
	}

	// apply the options
	for _, opt := range opts {
		opt(i)
	}

	return i
}

func (i *Index) GetLastIndexedTime() time.Time {
	return i.lastIndexed
}

func (i *Index) GetTopicByIdentifier(identifier string) *model.Topic {
	return i.topicsByIdentifier[identifier]
}

func (i *Index) GetArticleByIdentifier(topicIdentidier, identifier string) *model.Article {
	if topicArticles, ok := i.articlesByIdentifier[topicIdentidier]; ok {
		return topicArticles[identifier]
	}

	return nil
}

// GetTopics returns all the indexed topics
func (i *Index) GetAllTopics() []*model.Topic {
	topics := make([]*model.Topic, 0, len(i.topicsByIdentifier))
	for _, topic := range i.topicsByIdentifier {
		topics = append(topics, topic)
	}

	return topics
}

func (i *Index) GetAllArticlesForTopic(topicIdentifier string) []*model.Article {
	if _, ok := i.articlesByIdentifier[topicIdentifier]; !ok {
		return []*model.Article{}
	}

	articles := make([]*model.Article, 0, len(i.articlesByIdentifier[topicIdentifier]))
	for _, article := range i.articlesByIdentifier[topicIdentifier] {
		articles = append(articles, article)
	}

	return articles
}

// GetURIForFile returns the URI used by the file at the given path
func (i *Index) GetURIForFile(filepath string) string {
	if uri, ok := i.urisByFilepath[filepath]; ok {
		return uri
	}
	return ""
}

func (i *Index) GetRecentArticles(limit int) []*model.Article {
	if limit > len(i.articlesByTime) {
		limit = len(i.articlesByTime)
	}
	return i.articlesByTime[:limit]
}

func (i *Index) Reindex() {
	startTime := time.Now()

	topics := i.database.GetAllTopics()
	articles := i.database.GetAllArticles()

	// Perform indexing
	i.indexTopicsByIdentifier(topics)
	i.indexArticlesByIdentifier(articles)
	i.indexArticlesByTime(articles)
	i.indexArticlesByURI(articles)
	i.indexByURIsByFilepath(topics, articles)

	// Record the time
	i.lastIndexed = startTime

	i.metrics.Indexed(startTime, len(topics), len(articles))
}

func (i *Index) indexArticlesByTime(articles []*model.Article) {
	i.articlesByTime = []*model.Article{}

	for _, article := range articles {
		if !article.IsPublished() {
			continue
		}
		i.articlesByTime = append(i.articlesByTime, article)
	}

	sort.Slice(i.articlesByTime, func(x, y int) bool {
		return i.articlesByTime[y].PublishedAt < i.articlesByTime[x].PublishedAt
	})
}

func (i *Index) indexTopicsByIdentifier(topics []*model.Topic) {
	i.topicsByIdentifier = make(map[string]*model.Topic, len(topics))
	for _, topic := range topics {
		i.topicsByIdentifier[topic.Slug] = topic
	}
}

func (i *Index) indexArticlesByIdentifier(articles []*model.Article) {
	i.articlesByIdentifier = make(map[string]map[string]*model.Article)
	for _, article := range articles {
		if _, ok := i.articlesByIdentifier[article.TopicSlug]; !ok {
			i.articlesByIdentifier[article.TopicSlug] = make(map[string]*model.Article)
		}
		i.articlesByIdentifier[article.TopicSlug][article.Slug] = article
	}
}

// indexURIs stores entries by their URI
func (i *Index) indexArticlesByURI(articles []*model.Article) {
	i.articlesByURI = make(map[string]*model.Article, len(articles))
	for _, article := range articles {
		i.articlesByURI[strings.TrimLeft(filepath.Join(article.TopicSlug, article.Slug), "/")] = article
	}
}

// indexFilePaths indexes entries by their filepath on disk
func (i *Index) indexByURIsByFilepath(topics []*model.Topic, articles []*model.Article) {
	i.urisByFilepath = make(map[string]string, len(topics)+len(articles))
	for _, topic := range topics {
		i.urisByFilepath[topic.FilePath] = topic.URI
	}

	for _, article := range articles {
		i.urisByFilepath[article.FilePath] = article.URI
	}
}
