package reading_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wamphlett/blog-server/pkg/reading"
)

func newReader() *reading.Reader {
	return reading.New(nil, "", "", &MockMetrics{})
}

// --- LoadArticleFromFile ---

// verifies that all recognised header fields (title, description, slug, published, updated, hidden, priority, image, custom metadata) are parsed and mapped onto the Article struct
func TestLoadArticleFromFile_FullHeaders(t *testing.T) {
	r := newReader()
	article := r.LoadArticleFromFile("../../test/testdata/content/topic-one/article-with-headers.md", "go")

	require.NotNil(t, article)
	assert.Equal(t, "My Article", article.Title)
	assert.Equal(t, "A test article", article.Description)
	assert.Equal(t, "custom-slug", article.Slug)
	assert.Equal(t, "go", article.TopicSlug)
	assert.Equal(t, "/go/custom-slug", article.URI)
	assert.True(t, article.Hidden)
	assert.Equal(t, int64(5), article.Priority)
	assert.Equal(t, "cover.jpg", article.Image)
	assert.Equal(t, "custom_value", article.Metadata["custom_key"])

	expectedPublished, _ := time.Parse("2006-01-02", "2024-01-15")
	assert.Equal(t, expectedPublished.Unix(), article.PublishedAt)

	expectedUpdated, _ := time.Parse("2006-01-02", "2024-06-01")
	assert.Equal(t, expectedUpdated.Unix(), article.UpdatedAt)
}

// verifies that when no header block is present the slug and title fall back to the filename (without extension)
func TestLoadArticleFromFile_NoHeaders_DefaultsToFilename(t *testing.T) {
	r := newReader()
	article := r.LoadArticleFromFile("../../test/testdata/content/topic-one/no-headers.md", "go")

	require.NotNil(t, article)
	assert.Equal(t, "no-headers", article.Slug)
	assert.Equal(t, "no-headers", article.Title)
	assert.Equal(t, "/go/no-headers", article.URI)
	assert.Equal(t, int64(0), article.PublishedAt)
	assert.False(t, article.Hidden)
}

// verifies that the original file path is preserved on the Article so callers can later read the file
func TestLoadArticleFromFile_StoresFilePath(t *testing.T) {
	r := newReader()
	path := "../../test/testdata/content/topic-one/no-headers.md"
	article := r.LoadArticleFromFile(path, "go")

	assert.Equal(t, path, article.FilePath)
}

// verifies that the Metadata map is always initialised (never nil) even when no custom headers are present
func TestLoadArticleFromFile_MetadataInitialised(t *testing.T) {
	r := newReader()
	article := r.LoadArticleFromFile("../../test/testdata/content/topic-one/no-headers.md", "go")

	require.NotNil(t, article.Metadata)
}

// --- LoadTopicFromFile ---

// verifies that all recognised header fields are parsed and mapped onto the Topic struct, including custom metadata
func TestLoadTopicFromFile_FullHeaders(t *testing.T) {
	r := newReader()
	topic := r.LoadTopicFromFile("../../test/testdata/content/topic-with-headers/README.md")

	require.NotNil(t, topic)
	assert.Equal(t, "My Topic", topic.Title)
	assert.Equal(t, "A test topic", topic.Description)
	assert.Equal(t, "my-topic", topic.Slug)
	assert.Equal(t, "/my-topic", topic.URI)
	assert.False(t, topic.Hidden)
	assert.Equal(t, int64(3), topic.Priority)
	assert.Equal(t, "topic.jpg", topic.Image)
	assert.Equal(t, "topic_value", topic.Metadata["custom_field"])

	expectedPublished, _ := time.Parse("2006-01-02", "2024-03-20")
	assert.Equal(t, expectedPublished.Unix(), topic.PublishedAt)

	expectedUpdated, _ := time.Parse("2006-01-02", "2024-06-01")
	assert.Equal(t, expectedUpdated.Unix(), topic.UpdatedAt)
}

// verifies that when no slug header is present the slug falls back to the parent directory name
func TestLoadTopicFromFile_NoSlugHeader_DefaultsToDirectoryName(t *testing.T) {
	r := newReader()
	topic := r.LoadTopicFromFile("../../test/testdata/content/topic-one/README.md")

	require.NotNil(t, topic)
	assert.Equal(t, "topic-one", topic.Slug)
	assert.Equal(t, "/topic-one", topic.URI)
}

// verifies that the original file path is preserved on the Topic so callers can later read the file
func TestLoadTopicFromFile_StoresFilePath(t *testing.T) {
	r := newReader()
	path := "../../test/testdata/content/topic-one/README.md"
	topic := r.LoadTopicFromFile(path)

	assert.Equal(t, path, topic.FilePath)
}

// verifies that the Metadata map is always initialised (never nil) even when no custom headers are present
func TestLoadTopicFromFile_MetadataInitialised(t *testing.T) {
	r := newReader()
	topic := r.LoadTopicFromFile("../../test/testdata/content/topic-one/README.md")

	require.NotNil(t, topic.Metadata)
}
