package serving

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wamphlett/blog-server/pkg/model"
)

// --- mocks ---

type mockIndex struct {
	topics   []*model.Topic
	articles map[string][]*model.Article
}

func (m *mockIndex) GetLastIndexedTime() time.Time { return time.Unix(1000, 0) }

func (m *mockIndex) GetAllTopics() []*model.Topic { return m.topics }

func (m *mockIndex) GetTopicByIdentifier(id string) *model.Topic {
	for _, t := range m.topics {
		if t.Slug == id {
			return t
		}
	}
	return nil
}

func (m *mockIndex) GetArticleByIdentifier(topicID, id string) *model.Article {
	for _, a := range m.articles[topicID] {
		if a.Slug == id {
			return a
		}
	}
	return nil
}

func (m *mockIndex) GetAllArticlesForTopic(topicID string) []*model.Article {
	return m.articles[topicID]
}

func (m *mockIndex) GetRecentArticles(limit int) []*model.Article {
	all := []*model.Article{}
	for _, articles := range m.articles {
		all = append(all, articles...)
	}
	if limit < len(all) {
		return all[:limit]
	}
	return all
}

func (m *mockIndex) GetArticlesInSeries(series string) []*model.Article {
	matches := []*model.Article{}
	for _, articles := range m.articles {
		for _, a := range articles {
			if a.Series() == series && !a.Hidden {
				matches = append(matches, a)
			}
		}
	}
	sort.Slice(matches, func(x, y int) bool {
		if (matches[x].PublishedAt == 0) != (matches[y].PublishedAt == 0) {
			return matches[y].PublishedAt == 0
		}
		return matches[x].PublishedAt < matches[y].PublishedAt
	})
	return matches
}

type mockFileReader struct {
	content string
	err     error
}

func (m *mockFileReader) ReadFileAsHTML(_ context.Context, _ string) (string, error) {
	return m.content, m.err
}

type mockServingMetrics struct{}

func (m *mockServingMetrics) Request(_ context.Context, _ string, _ int, _ time.Time) {}

// --- helpers ---

func newTestServer(idx *mockIndex, reader *mockFileReader) *Server {
	return New(reader, idx, "", "", "", &mockServingMetrics{})
}

func doRequest(s *Server, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

// --- status ---

// verifies that /status returns 200 with ready:true and the last indexed timestamp from the index
func TestStatus_ReturnsOK(t *testing.T) {
	s := newTestServer(&mockIndex{}, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/status")

	require.Equal(t, http.StatusOK, w.Code)

	var resp StatusResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Ready)
	assert.Equal(t, int64(1000), resp.LastIndexed)
}

// --- listTopics ---

// verifies that /topics returns all indexed topics with their published article counts
func TestListTopics_ReturnsList(t *testing.T) {
	idx := &mockIndex{
		topics: []*model.Topic{
			{Slug: "go", Title: "Go", PublishedAt: time.Now().Add(-time.Hour).Unix()},
		},
		articles: map[string][]*model.Article{
			"go": {
				{TopicSlug: "go", Slug: "goroutines", PublishedAt: time.Now().Add(-time.Hour).Unix()},
				{TopicSlug: "go", Slug: "channels", PublishedAt: time.Now().Add(-time.Hour).Unix()},
			},
		},
	}
	s := newTestServer(idx, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/topics")

	require.Equal(t, http.StatusOK, w.Code)

	var resp ListTopicsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Topics, 1)
	assert.Equal(t, "go", resp.Topics[0].Slug)
	assert.Equal(t, 2, resp.Topics[0].PublishedArticleCount)
}

// verifies that /topics returns an empty list when no topics are indexed
func TestListTopics_EmptyIndex(t *testing.T) {
	s := newTestServer(&mockIndex{}, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/topics")

	require.Equal(t, http.StatusOK, w.Code)

	var resp ListTopicsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Empty(t, resp.Topics)
}

// --- getTopic ---

// verifies that /topics/{topic} returns the topic metadata and its rendered HTML content
func TestGetTopic_Found(t *testing.T) {
	idx := &mockIndex{
		topics: []*model.Topic{
			{Slug: "go", Title: "Go", FilePath: "/content/go/README.md"},
		},
		articles: map[string][]*model.Article{},
	}
	s := newTestServer(idx, &mockFileReader{content: "<h1>Go</h1>"})
	w := doRequest(s, http.MethodGet, "/topics/go")

	require.Equal(t, http.StatusOK, w.Code)

	var resp GetTopicResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "go", resp.Slug)
	assert.Equal(t, "<h1>Go</h1>", resp.Html)
}

// verifies that /topics/{topic} returns 404 with an error message when the topic slug is not indexed
func TestGetTopic_NotFound(t *testing.T) {
	s := newTestServer(&mockIndex{}, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/topics/missing")

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "not found", resp.Message)
}

// --- listArticles ---

// verifies that /topics/{topic}/articles returns all articles for the given topic
func TestListArticles_Found(t *testing.T) {
	idx := &mockIndex{
		topics: []*model.Topic{
			{Slug: "go", Title: "Go"},
		},
		articles: map[string][]*model.Article{
			"go": {
				{TopicSlug: "go", Slug: "goroutines", Title: "Goroutines"},
			},
		},
	}
	s := newTestServer(idx, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/topics/go/articles")

	require.Equal(t, http.StatusOK, w.Code)

	var resp ListArticlesResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Articles, 1)
	assert.Equal(t, "goroutines", resp.Articles[0].Slug)
}

// verifies that /topics/{topic}/articles returns 404 when the topic slug is not indexed
func TestListArticles_UnknownTopic(t *testing.T) {
	s := newTestServer(&mockIndex{}, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/topics/missing/articles")

	require.Equal(t, http.StatusNotFound, w.Code)
}

// --- getArticle ---

// verifies that /topics/{topic}/articles/{article} returns the article metadata, topic slug, and rendered HTML
func TestGetArticle_Found(t *testing.T) {
	idx := &mockIndex{
		topics: []*model.Topic{
			{Slug: "go", Title: "Go"},
		},
		articles: map[string][]*model.Article{
			"go": {
				{TopicSlug: "go", Slug: "goroutines", Title: "Goroutines", FilePath: "/content/go/goroutines.md"},
			},
		},
	}
	s := newTestServer(idx, &mockFileReader{content: "<h1>Goroutines</h1>"})
	w := doRequest(s, http.MethodGet, "/topics/go/articles/goroutines")

	require.Equal(t, http.StatusOK, w.Code)

	var resp GetArticleResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "goroutines", resp.Slug)
	assert.Equal(t, "go", resp.TopicSlug)
	assert.Equal(t, "<h1>Goroutines</h1>", resp.Html)
}

// verifies that /topics/{topic}/articles/{article} returns 404 when the topic slug is not indexed
func TestGetArticle_UnknownTopic(t *testing.T) {
	s := newTestServer(&mockIndex{}, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/topics/missing/articles/goroutines")

	require.Equal(t, http.StatusNotFound, w.Code)
}

// verifies that /topics/{topic}/articles/{article} returns 404 when the article slug is not indexed under the topic
func TestGetArticle_UnknownArticle(t *testing.T) {
	idx := &mockIndex{
		topics: []*model.Topic{
			{Slug: "go"},
		},
		articles: map[string][]*model.Article{},
	}
	s := newTestServer(idx, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/topics/go/articles/missing")

	require.Equal(t, http.StatusNotFound, w.Code)
}

// verifies that articles in a series include the series payload with all parts
// (published and unpublished) ordered by published date with a published flag,
// and articles without a series omit it
func TestGetArticle_Series(t *testing.T) {
	past := time.Now().Add(-time.Hour).Unix()
	idx := &mockIndex{
		topics: []*model.Topic{
			{Slug: "go"},
			{Slug: "rust"},
		},
		articles: map[string][]*model.Article{
			"go": {
				{TopicSlug: "go", Slug: "part-1", Title: "Part 1", PublishedAt: past - 200, Metadata: map[string]string{"series": "my-series"}},
				{TopicSlug: "go", Slug: "part-2", Title: "Part 2", PublishedAt: past - 100, Metadata: map[string]string{"series": "my-series"}},
				{TopicSlug: "go", Slug: "part-4", Title: "Part 4", PublishedAt: 0, Metadata: map[string]string{"series": "my-series"}},
				{TopicSlug: "go", Slug: "standalone", Title: "Standalone", PublishedAt: past, Metadata: map[string]string{}},
			},
			"rust": {
				{TopicSlug: "rust", Slug: "part-3", Title: "Part 3", PublishedAt: past, Metadata: map[string]string{"series": "my-series"}},
			},
		},
	}
	s := newTestServer(idx, &mockFileReader{content: "<p>content</p>"})

	// article in a series returns the full series, ordered, across topics,
	// with unpublished parts included and flagged
	w := doRequest(s, http.MethodGet, "/topics/go/articles/part-2")
	require.Equal(t, http.StatusOK, w.Code)

	var resp GetArticleResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotNil(t, resp.Series)
	assert.Equal(t, "my-series", resp.Series.Name)
	require.Len(t, resp.Series.Articles, 4)
	assert.Equal(t, "part-1", resp.Series.Articles[0].Slug)
	assert.Equal(t, "part-2", resp.Series.Articles[1].Slug)
	assert.Equal(t, "part-3", resp.Series.Articles[2].Slug)
	assert.Equal(t, "part-4", resp.Series.Articles[3].Slug)
	assert.Equal(t, "/topics/rust/articles/part-3", resp.Series.Articles[2].URL)
	assert.True(t, resp.Series.Articles[0].Published)
	assert.True(t, resp.Series.Articles[2].Published)
	assert.Equal(t, false, resp.Series.Articles[3].Published)

	// article without a series omits the payload
	w = doRequest(s, http.MethodGet, "/topics/go/articles/standalone")
	require.Equal(t, http.StatusOK, w.Code)

	var standalone GetArticleResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&standalone))
	assert.Nil(t, standalone.Series)
}

// --- getRecent ---

// verifies that /recent returns at most 3 articles when no limit query param is provided
func TestGetRecent_DefaultLimit(t *testing.T) {
	idx := &mockIndex{
		topics: []*model.Topic{
			{Slug: "go"},
		},
		articles: map[string][]*model.Article{
			"go": {
				{TopicSlug: "go", Slug: "a"},
				{TopicSlug: "go", Slug: "b"},
				{TopicSlug: "go", Slug: "c"},
				{TopicSlug: "go", Slug: "d"},
			},
		},
	}
	s := newTestServer(idx, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/recent")

	require.Equal(t, http.StatusOK, w.Code)

	var resp ListArticlesResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp.Articles, 3)
}

// verifies that /recent?limit=N honours the limit query parameter
func TestGetRecent_CustomLimit(t *testing.T) {
	idx := &mockIndex{
		topics: []*model.Topic{
			{Slug: "go"},
		},
		articles: map[string][]*model.Article{
			"go": {
				{TopicSlug: "go", Slug: "a"},
				{TopicSlug: "go", Slug: "b"},
				{TopicSlug: "go", Slug: "c"},
			},
		},
	}
	s := newTestServer(idx, &mockFileReader{})
	w := doRequest(s, http.MethodGet, "/recent?limit=1")

	require.Equal(t, http.StatusOK, w.Code)

	var resp ListArticlesResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp.Articles, 1)
}

// --- URL building ---

// verifies that topic responses carry the correct self URL and articles collection URL
func TestConvertTopic_URLPattern(t *testing.T) {
	topic := &model.Topic{Slug: "go", Title: "Go"}
	result := convertTopic(topic, nil)

	assert.Equal(t, "/topics/go", result.URL)
	assert.Equal(t, "/topics/go/articles", result.ArticleURL)
}

// verifies that article responses carry the correct nested URL and propagate the topic slug
func TestConvertArticle_URLPattern(t *testing.T) {
	topic := &model.Topic{Slug: "go"}
	article := &model.Article{Slug: "goroutines", TopicSlug: "go"}
	result := convertArticle(topic, article)

	assert.Equal(t, "/topics/go/articles/goroutines", result.URL)
	assert.Equal(t, "go", result.TopicSlug)
}

// verifies that only articles that are past their publish date and not hidden count towards PublishedArticleCount
func TestConvertTopic_PublishedArticleCount(t *testing.T) {
	topic := &model.Topic{Slug: "go"}
	past := time.Now().Add(-time.Hour).Unix()
	future := time.Now().Add(time.Hour).Unix()

	articles := []*model.Article{
		{PublishedAt: past, Hidden: false},
		{PublishedAt: past, Hidden: true},
		{PublishedAt: future, Hidden: false},
		{PublishedAt: 0},
	}

	result := convertTopic(topic, articles)
	assert.Equal(t, 1, result.PublishedArticleCount)
}
