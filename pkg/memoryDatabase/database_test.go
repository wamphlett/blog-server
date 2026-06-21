package memorydatabase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	memorydatabase "github.com/wamphlett/blog-server/pkg/memoryDatabase"
	"github.com/wamphlett/blog-server/pkg/model"
)

// verifies that stored topics are all returned by GetAllTopics
func TestStoreTopic_GetAllTopics(t *testing.T) {
	db := memorydatabase.New()

	db.StoreTopic(&model.Topic{Slug: "go", Title: "Go"})
	db.StoreTopic(&model.Topic{Slug: "rust", Title: "Rust"})

	topics := db.GetAllTopics()
	require.Len(t, topics, 2)

	slugs := map[string]bool{}
	for _, t := range topics {
		slugs[t.Slug] = true
	}
	assert.True(t, slugs["go"])
	assert.True(t, slugs["rust"])
}

// verifies that storing a topic with a duplicate slug replaces the previous entry
func TestStoreTopic_OverwritesExistingSlug(t *testing.T) {
	db := memorydatabase.New()

	db.StoreTopic(&model.Topic{Slug: "go", Title: "Old"})
	db.StoreTopic(&model.Topic{Slug: "go", Title: "New"})

	topics := db.GetAllTopics()
	require.Len(t, topics, 1)
	assert.Equal(t, "New", topics[0].Title)
}

// verifies that GetAllTopics returns an empty slice when no topics have been stored
func TestGetAllTopics_EmptyDatabase(t *testing.T) {
	db := memorydatabase.New()
	require.Empty(t, db.GetAllTopics())
}

// verifies that articles stored across multiple topics are all returned by GetAllArticles
func TestStoreArticle_GetAllArticles(t *testing.T) {
	db := memorydatabase.New()

	db.StoreArticle(&model.Article{TopicSlug: "go", Slug: "goroutines"})
	db.StoreArticle(&model.Article{TopicSlug: "go", Slug: "channels"})
	db.StoreArticle(&model.Article{TopicSlug: "rust", Slug: "ownership"})

	articles := db.GetAllArticles()
	require.Len(t, articles, 3)
}

// verifies that storing an article with a duplicate slug within the same topic replaces the previous entry
func TestStoreArticle_OverwritesExistingSlug(t *testing.T) {
	db := memorydatabase.New()

	db.StoreArticle(&model.Article{TopicSlug: "go", Slug: "goroutines", Title: "Old"})
	db.StoreArticle(&model.Article{TopicSlug: "go", Slug: "goroutines", Title: "New"})

	articles := db.GetAllArticles()
	require.Len(t, articles, 1)
	assert.Equal(t, "New", articles[0].Title)
}

// verifies that only articles belonging to the requested topic slug are returned
func TestGetAllArticlesForTopic(t *testing.T) {
	db := memorydatabase.New()

	db.StoreArticle(&model.Article{TopicSlug: "go", Slug: "goroutines"})
	db.StoreArticle(&model.Article{TopicSlug: "go", Slug: "channels"})
	db.StoreArticle(&model.Article{TopicSlug: "rust", Slug: "ownership"})

	goArticles := db.GetAllArticlesForTopic("go")
	require.Len(t, goArticles, 2)

	rustArticles := db.GetAllArticlesForTopic("rust")
	require.Len(t, rustArticles, 1)
}

// verifies that querying articles for a topic that does not exist returns an empty slice rather than panicking
func TestGetAllArticlesForTopic_UnknownTopic(t *testing.T) {
	db := memorydatabase.New()
	require.Empty(t, db.GetAllArticlesForTopic("nope"))
}

// verifies that GetAllArticles returns an empty slice when no articles have been stored
func TestGetAllArticles_EmptyDatabase(t *testing.T) {
	db := memorydatabase.New()
	require.Empty(t, db.GetAllArticles())
}
