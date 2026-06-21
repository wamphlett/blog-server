package indexing_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wamphlett/blog-server/pkg/indexing"
	"github.com/wamphlett/blog-server/pkg/model"
)

type mockDB struct {
	topics   []*model.Topic
	articles []*model.Article
}

func (m *mockDB) GetAllTopics() []*model.Topic    { return m.topics }
func (m *mockDB) GetAllArticles() []*model.Article { return m.articles }

type mockMetrics struct{}

func (m *mockMetrics) Indexed(_ context.Context, _ time.Time, _, _ int) {}

func newTestIndex(topics []*model.Topic, articles []*model.Article, opts ...indexing.Option) *indexing.Index {
	db := &mockDB{topics: topics, articles: articles}
	idx := indexing.NewIndex(db, &mockMetrics{}, opts...)
	idx.Reindex()
	return idx
}

// verifies that topics can be looked up by slug after a reindex, and returns nil for unknown slugs
func TestGetTopicByIdentifier(t *testing.T) {
	topics := []*model.Topic{
		{Slug: "go", Title: "Go"},
		{Slug: "rust", Title: "Rust"},
	}
	idx := newTestIndex(topics, nil)

	assert.Equal(t, "Go", idx.GetTopicByIdentifier("go").Title)
	assert.Equal(t, "Rust", idx.GetTopicByIdentifier("rust").Title)
	assert.Nil(t, idx.GetTopicByIdentifier("missing"))
}

// verifies that all topics from the database are available in the index after a reindex
func TestGetAllTopics(t *testing.T) {
	topics := []*model.Topic{
		{Slug: "go"},
		{Slug: "rust"},
	}
	idx := newTestIndex(topics, nil)

	result := idx.GetAllTopics()
	require.Len(t, result, 2)
}

// verifies that articles can be looked up by topic slug and article slug, and returns nil for unknown combinations
func TestGetArticleByIdentifier(t *testing.T) {
	articles := []*model.Article{
		{TopicSlug: "go", Slug: "goroutines"},
		{TopicSlug: "go", Slug: "channels"},
	}
	idx := newTestIndex(nil, articles)

	assert.Equal(t, "goroutines", idx.GetArticleByIdentifier("go", "goroutines").Slug)
	assert.Nil(t, idx.GetArticleByIdentifier("go", "missing"))
	assert.Nil(t, idx.GetArticleByIdentifier("missing", "goroutines"))
}

// verifies that only articles belonging to the requested topic slug are returned, and unknown topics return empty
func TestGetAllArticlesForTopic(t *testing.T) {
	articles := []*model.Article{
		{TopicSlug: "go", Slug: "goroutines"},
		{TopicSlug: "go", Slug: "channels"},
		{TopicSlug: "rust", Slug: "ownership"},
	}
	idx := newTestIndex(nil, articles)

	goArticles := idx.GetAllArticlesForTopic("go")
	require.Len(t, goArticles, 2)

	rustArticles := idx.GetAllArticlesForTopic("rust")
	require.Len(t, rustArticles, 1)

	require.Empty(t, idx.GetAllArticlesForTopic("missing"))
}

// verifies that the index maps file paths to their URIs for both topics and articles, and returns empty string for unknown paths
func TestGetURIForFile(t *testing.T) {
	topics := []*model.Topic{
		{Slug: "go", FilePath: "/content/go/README.md", URI: "/go"},
	}
	articles := []*model.Article{
		{TopicSlug: "go", Slug: "goroutines", FilePath: "/content/go/goroutines.md", URI: "/go/goroutines"},
	}
	idx := newTestIndex(topics, articles)

	assert.Equal(t, "/go", idx.GetURIForFile("/content/go/README.md"))
	assert.Equal(t, "/go/goroutines", idx.GetURIForFile("/content/go/goroutines.md"))
	assert.Equal(t, "", idx.GetURIForFile("/content/missing.md"))
}

// verifies that recent articles are returned newest-first by published date
func TestGetRecentArticles_OrderedByPublishedAtDesc(t *testing.T) {
	articles := []*model.Article{
		{TopicSlug: "go", Slug: "oldest", PublishedAt: 1000},
		{TopicSlug: "go", Slug: "newest", PublishedAt: 3000},
		{TopicSlug: "go", Slug: "middle", PublishedAt: 2000},
	}
	idx := newTestIndex(nil, articles)

	recent := idx.GetRecentArticles(3)
	require.Len(t, recent, 3)
	assert.Equal(t, "newest", recent[0].Slug)
	assert.Equal(t, "middle", recent[1].Slug)
	assert.Equal(t, "oldest", recent[2].Slug)
}

// verifies that the limit parameter caps the number of articles returned
func TestGetRecentArticles_LimitRespected(t *testing.T) {
	articles := []*model.Article{
		{TopicSlug: "go", Slug: "a", PublishedAt: 1000},
		{TopicSlug: "go", Slug: "b", PublishedAt: 2000},
		{TopicSlug: "go", Slug: "c", PublishedAt: 3000},
	}
	idx := newTestIndex(nil, articles)

	require.Len(t, idx.GetRecentArticles(2), 2)
	require.Len(t, idx.GetRecentArticles(1), 1)
}

// verifies that requesting more articles than exist returns all available articles without panicking
func TestGetRecentArticles_LimitExceedsCount(t *testing.T) {
	articles := []*model.Article{
		{TopicSlug: "go", Slug: "a", PublishedAt: 1000},
	}
	idx := newTestIndex(nil, articles)

	require.Len(t, idx.GetRecentArticles(100), 1)
}

// verifies that future-dated, hidden, and zero-date articles are excluded from recent results
func TestGetRecentArticles_ExcludesUnpublished(t *testing.T) {
	now := time.Now().Unix()
	future := time.Now().Add(time.Hour).Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "published", PublishedAt: now - 3600},
		{TopicSlug: "go", Slug: "future", PublishedAt: future},
		{TopicSlug: "go", Slug: "hidden", PublishedAt: now - 3600, Hidden: true},
		{TopicSlug: "go", Slug: "no-date", PublishedAt: 0},
	}
	idx := newTestIndex(nil, articles)

	recent := idx.GetRecentArticles(10)
	require.Len(t, recent, 1)
	assert.Equal(t, "published", recent[0].Slug)
}

// verifies that GetLastIndexedTime reflects the time at which Reindex was called
func TestReindex_UpdatesLastIndexedTime(t *testing.T) {
	before := time.Now()
	idx := newTestIndex(nil, nil)
	after := time.Now()

	lastIndexed := idx.GetLastIndexedTime()
	assert.True(t, !lastIndexed.Before(before))
	assert.True(t, !lastIndexed.After(after))
}
