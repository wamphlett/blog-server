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

// verifies that a topic's articles are always ordered newest published first, with
// undated articles last and ties broken by slug for a deterministic order
func TestGetAllArticlesForTopic_OrderedByPublishedAtDesc(t *testing.T) {
	now := time.Now().Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "oldest", PublishedAt: now - 2000},
		{TopicSlug: "go", Slug: "undated-b", PublishedAt: 0},
		{TopicSlug: "go", Slug: "newest", PublishedAt: now - 1000},
		{TopicSlug: "go", Slug: "undated-a", PublishedAt: 0},
	}
	idx := newTestIndex(nil, articles)

	goArticles := idx.GetAllArticlesForTopic("go")
	require.Len(t, goArticles, 4)
	assert.Equal(t, "newest", goArticles[0].Slug)
	assert.Equal(t, "oldest", goArticles[1].Slug)
	assert.Equal(t, "undated-a", goArticles[2].Slug)
	assert.Equal(t, "undated-b", goArticles[3].Slug)
}

// verifies that undated articles are ordered by creation date (newest first) rather
// than falling back straight to slug, and that slug is still used as a final
// tiebreak when creation date is also equal or unknown
func TestGetAllArticlesForTopic_UndatedOrderedByCreatedAt(t *testing.T) {
	now := time.Now().Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "z-oldest-draft", PublishedAt: 0, CreatedAt: now - 3000},
		{TopicSlug: "go", Slug: "a-newest-draft", PublishedAt: 0, CreatedAt: now - 1000},
		{TopicSlug: "go", Slug: "b-no-created-at", PublishedAt: 0, CreatedAt: 0},
		{TopicSlug: "go", Slug: "a-no-created-at", PublishedAt: 0, CreatedAt: 0},
	}
	idx := newTestIndex(nil, articles)

	goArticles := idx.GetAllArticlesForTopic("go")
	require.Len(t, goArticles, 4)
	assert.Equal(t, "a-newest-draft", goArticles[0].Slug)
	assert.Equal(t, "z-oldest-draft", goArticles[1].Slug)
	assert.Equal(t, "a-no-created-at", goArticles[2].Slug)
	assert.Equal(t, "b-no-created-at", goArticles[3].Slug)
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

// verifies that staging mode includes future-dated and undated articles in recent
// results, but still excludes hidden articles
func TestGetRecentArticles_StagingModeIncludesUnpublished(t *testing.T) {
	now := time.Now().Unix()
	future := time.Now().Add(time.Hour).Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "published", PublishedAt: now - 3600},
		{TopicSlug: "go", Slug: "future", PublishedAt: future},
		{TopicSlug: "go", Slug: "hidden", PublishedAt: now - 3600, Hidden: true},
		{TopicSlug: "go", Slug: "no-date", PublishedAt: 0},
	}
	idx := newTestIndex(nil, articles, indexing.WithStagingMode(true))

	recent := idx.GetRecentArticles(10)
	slugs := make([]string, len(recent))
	for i, a := range recent {
		slugs[i] = a.Slug
	}
	assert.ElementsMatch(t, []string{"published", "future", "no-date"}, slugs)
}

// verifies that in staging mode, undated articles in the recent feed are ordered by
// creation date (newest first) instead of scattering unpredictably
func TestGetRecentArticles_StagingModeUndatedOrderedByCreatedAt(t *testing.T) {
	now := time.Now().Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "older-draft", PublishedAt: 0, CreatedAt: now - 2000},
		{TopicSlug: "go", Slug: "newer-draft", PublishedAt: 0, CreatedAt: now - 1000},
	}
	idx := newTestIndex(nil, articles, indexing.WithStagingMode(true))

	recent := idx.GetRecentArticles(10)
	require.Len(t, recent, 2)
	assert.Equal(t, "newer-draft", recent[0].Slug)
	assert.Equal(t, "older-draft", recent[1].Slug)
}

// verifies that articles in a series are returned ordered by published date (oldest
// first), grouped globally across topics, with case-insensitive series names
func TestGetArticlesInSeries(t *testing.T) {
	past := time.Now().Add(-time.Hour).Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "part-2", PublishedAt: past - 100, Metadata: map[string]string{"series": "My Series"}},
		{TopicSlug: "go", Slug: "part-1", PublishedAt: past - 200, Metadata: map[string]string{"series": "my series"}},
		{TopicSlug: "rust", Slug: "part-3", PublishedAt: past, Metadata: map[string]string{"series": "my series"}},
		{TopicSlug: "go", Slug: "other", PublishedAt: past, Metadata: map[string]string{"series": "other series"}},
		{TopicSlug: "go", Slug: "no-series", PublishedAt: past, Metadata: map[string]string{}},
	}
	idx := newTestIndex(nil, articles)

	series := idx.GetArticlesInSeries("my series")
	require.Len(t, series, 3)
	assert.Equal(t, "part-1", series[0].Slug)
	assert.Equal(t, "part-2", series[1].Slug)
	assert.Equal(t, "part-3", series[2].Slug)

	// case-insensitive lookup
	require.Len(t, idx.GetArticlesInSeries("MY SERIES"), 3)

	require.Len(t, idx.GetArticlesInSeries("other series"), 1)
	require.Empty(t, idx.GetArticlesInSeries("missing"))
}

// verifies that unpublished articles are included in series (future-dated in date
// order, undated last) while hidden articles are excluded
func TestGetArticlesInSeries_IncludesUnpublished(t *testing.T) {
	past := time.Now().Add(-time.Hour).Unix()
	future := time.Now().Add(time.Hour).Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "draft", PublishedAt: 0, Metadata: map[string]string{"series": "s"}},
		{TopicSlug: "go", Slug: "future", PublishedAt: future, Metadata: map[string]string{"series": "s"}},
		{TopicSlug: "go", Slug: "published", PublishedAt: past, Metadata: map[string]string{"series": "s"}},
		{TopicSlug: "go", Slug: "hidden", PublishedAt: past, Hidden: true, Metadata: map[string]string{"series": "s"}},
	}
	idx := newTestIndex(nil, articles)

	series := idx.GetArticlesInSeries("s")
	require.Len(t, series, 3)
	assert.Equal(t, "published", series[0].Slug)
	assert.Equal(t, "future", series[1].Slug)
	assert.Equal(t, "draft", series[2].Slug)
}

// verifies that articles sharing the same published date (most commonly a group of
// undated, upcoming articles) are ordered by priority (highest first), then by slug,
// so the order is deterministic instead of depending on database iteration order
func TestGetArticlesInSeries_TiesBrokenByPriorityThenSlug(t *testing.T) {
	articles := []*model.Article{
		{TopicSlug: "go", Slug: "no-priority-b", PublishedAt: 0, Priority: 0, Metadata: map[string]string{"series": "s"}},
		{TopicSlug: "go", Slug: "high-priority", PublishedAt: 0, Priority: 10, Metadata: map[string]string{"series": "s"}},
		{TopicSlug: "go", Slug: "no-priority-a", PublishedAt: 0, Priority: 0, Metadata: map[string]string{"series": "s"}},
		{TopicSlug: "go", Slug: "mid-priority", PublishedAt: 0, Priority: 5, Metadata: map[string]string{"series": "s"}},
	}
	idx := newTestIndex(nil, articles)

	series := idx.GetArticlesInSeries("s")
	require.Len(t, series, 4)
	assert.Equal(t, "high-priority", series[0].Slug)
	assert.Equal(t, "mid-priority", series[1].Slug)
	assert.Equal(t, "no-priority-a", series[2].Slug)
	assert.Equal(t, "no-priority-b", series[3].Slug)
}

// verifies that priority is only used as a tiebreaker: when articles have distinct
// published dates, date order wins regardless of priority
func TestGetArticlesInSeries_DateOrderIgnoresPriorityWhenDatesDiffer(t *testing.T) {
	past := time.Now().Add(-time.Hour).Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "later-low-priority", PublishedAt: past, Priority: 0, Metadata: map[string]string{"series": "s"}},
		{TopicSlug: "go", Slug: "earlier-high-priority", PublishedAt: past - 100, Priority: 999, Metadata: map[string]string{"series": "s"}},
	}
	idx := newTestIndex(nil, articles)

	series := idx.GetArticlesInSeries("s")
	require.Len(t, series, 2)
	assert.Equal(t, "earlier-high-priority", series[0].Slug)
	assert.Equal(t, "later-low-priority", series[1].Slug)
}

// verifies that priority breaks ties between articles published on the exact same
// timestamp
func TestGetArticlesInSeries_SameDatePriorityTiebreak(t *testing.T) {
	same := time.Now().Add(-time.Hour).Unix()

	articles := []*model.Article{
		{TopicSlug: "go", Slug: "low-priority", PublishedAt: same, Priority: 1, Metadata: map[string]string{"series": "s"}},
		{TopicSlug: "go", Slug: "high-priority", PublishedAt: same, Priority: 2, Metadata: map[string]string{"series": "s"}},
	}
	idx := newTestIndex(nil, articles)

	series := idx.GetArticlesInSeries("s")
	require.Len(t, series, 2)
	assert.Equal(t, "high-priority", series[0].Slug)
	assert.Equal(t, "low-priority", series[1].Slug)
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
