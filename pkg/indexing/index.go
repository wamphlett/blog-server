package indexing

import (
	"context"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/wamphlett/blog-server/pkg/model"
	"github.com/wamphlett/blog-server/pkg/telemetry"
)

var tracer = otel.Tracer(telemetry.InstrumentationName)

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

// WithStagingMode controls whether unpublished (future-dated or undated)
// articles are included in the recent articles feed. Hidden articles are
// always excluded regardless of this setting.
func WithStagingMode(stagingMode bool) Option {
	return func(i *Index) {
		i.stagingMode = stagingMode
	}
}

// Metrics defines the metrics used by the index
type Metrics interface {
	Indexed(ctx context.Context, startTime time.Time, topicCount, articleCount int)
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
	articlesByTopic      map[string][]*model.Article
	articlesByTime       []*model.Article
	articlesByURI        map[string]*model.Article
	articlesBySeries     map[string][]*model.Article
	urisByFilepath       map[string]string

	// last indexed time
	lastIndexed time.Time

	// when true, unpublished articles are included in the recent articles feed
	stagingMode bool

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

// GetAllArticlesForTopic returns all articles for the given topic, ordered
// newest published first, with undated articles last.
func (i *Index) GetAllArticlesForTopic(topicIdentifier string) []*model.Article {
	if articles, ok := i.articlesByTopic[topicIdentifier]; ok {
		return articles
	}
	return []*model.Article{}
}

// GetURIForFile returns the URI used by the file at the given path
func (i *Index) GetURIForFile(filepath string) string {
	if uri, ok := i.urisByFilepath[filepath]; ok {
		return uri
	}
	return ""
}

// GetArticlesInSeries returns all articles in the given series, including
// unpublished ones (but not hidden ones), ordered by published date (oldest
// first) with undated articles last. Series names are case-insensitive.
func (i *Index) GetArticlesInSeries(series string) []*model.Article {
	if articles, ok := i.articlesBySeries[seriesKey(series)]; ok {
		return articles
	}
	return []*model.Article{}
}

func (i *Index) GetRecentArticles(limit int) []*model.Article {
	if limit > len(i.articlesByTime) {
		limit = len(i.articlesByTime)
	}
	return i.articlesByTime[:limit]
}

func (i *Index) Reindex() {
	ctx, span := tracer.Start(context.Background(), "index.Reindex")
	defer span.End()

	startTime := time.Now()
	slog.InfoContext(ctx, "reindexing")

	topics := i.database.GetAllTopics()
	articles := i.database.GetAllArticles()

	i.indexTopicsByIdentifier(topics)
	i.indexArticlesByIdentifier(articles)
	i.indexArticlesByTopic(articles)
	i.indexArticlesByTime(articles)
	i.indexArticlesByURI(articles)
	i.indexArticlesBySeries(articles)
	i.indexByURIsByFilepath(topics, articles)

	i.lastIndexed = startTime
	i.metrics.Indexed(ctx, startTime, len(topics), len(articles))

	slog.InfoContext(ctx, "reindex complete", "topics", len(topics), "articles", len(articles), "duration", time.Since(startTime))
}

func (i *Index) indexArticlesByTime(articles []*model.Article) {
	i.articlesByTime = []*model.Article{}

	for _, article := range articles {
		if article.Hidden {
			continue
		}
		if !i.stagingMode && !article.IsPublished() {
			continue
		}
		i.articlesByTime = append(i.articlesByTime, article)
	}

	sort.Slice(i.articlesByTime, func(x, y int) bool {
		return lessRecent(i.articlesByTime[x], i.articlesByTime[y])
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

// lessRecent orders articles newest published first, with undated articles
// last. Ties (most commonly a group of undated articles) are broken by
// creation date (newest first), then by slug as a final deterministic
// fallback when the creation date is also unknown or equal.
func lessRecent(a, b *model.Article) bool {
	if (a.PublishedAt == 0) != (b.PublishedAt == 0) {
		return b.PublishedAt == 0
	}
	if a.PublishedAt != b.PublishedAt {
		return a.PublishedAt > b.PublishedAt
	}
	if a.CreatedAt != b.CreatedAt {
		return a.CreatedAt > b.CreatedAt
	}
	return a.Slug < b.Slug
}

// indexArticlesByTopic groups articles by topic, ordered newest published
// first with undated articles last (see lessRecent).
func (i *Index) indexArticlesByTopic(articles []*model.Article) {
	i.articlesByTopic = make(map[string][]*model.Article)
	for _, article := range articles {
		i.articlesByTopic[article.TopicSlug] = append(i.articlesByTopic[article.TopicSlug], article)
	}

	for _, articles := range i.articlesByTopic {
		sort.Slice(articles, func(x, y int) bool {
			return lessRecent(articles[x], articles[y])
		})
	}
}

// indexURIs stores entries by their URI
func (i *Index) indexArticlesByURI(articles []*model.Article) {
	i.articlesByURI = make(map[string]*model.Article, len(articles))
	for _, article := range articles {
		i.articlesByURI[strings.TrimLeft(filepath.Join(article.TopicSlug, article.Slug), "/")] = article
	}
}

// indexArticlesBySeries groups articles by their series name, ordered by
// published date (oldest first) with undated articles last. Articles that
// share a date (most commonly a group of undated, upcoming articles) are
// then ordered by priority (highest first), then by slug so the order is
// fully deterministic regardless of database iteration order. Unpublished
// articles are included so upcoming parts can be surfaced; hidden articles
// are excluded.
func (i *Index) indexArticlesBySeries(articles []*model.Article) {
	i.articlesBySeries = make(map[string][]*model.Article)
	for _, article := range articles {
		series := article.Series()
		if series == "" || article.Hidden {
			continue
		}
		key := seriesKey(series)
		i.articlesBySeries[key] = append(i.articlesBySeries[key], article)
	}

	for _, articles := range i.articlesBySeries {
		sort.Slice(articles, func(x, y int) bool {
			a, b := articles[x], articles[y]

			// undated articles sort last
			if (a.PublishedAt == 0) != (b.PublishedAt == 0) {
				return b.PublishedAt == 0
			}
			if a.PublishedAt != b.PublishedAt {
				return a.PublishedAt < b.PublishedAt
			}
			if a.Priority != b.Priority {
				return a.Priority > b.Priority
			}
			return a.Slug < b.Slug
		})
	}
}

// seriesKey normalises a series name so lookups are case-insensitive
func seriesKey(series string) string {
	return strings.ToLower(strings.TrimSpace(series))
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
