package reading_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wamphlett/blog-server/pkg/reading"
)

type MockMetrics struct{}

func (m *MockMetrics) ParseFile(_ context.Context, startTime time.Time)    {}
func (m *MockMetrics) ParseHeaders(_ context.Context, startTime time.Time) {}

type MockIndex struct {
	uris map[string]string
}

func (m *MockIndex) GetURIForFile(path string) string {
	return m.uris[path]
}

// verifies that HTML comment property blocks (<!-- -->) are preserved verbatim in the rendered output
func TestReadsFileAsHTMLStripsProperties(t *testing.T) {
	reader := reading.New(nil, "", "", &MockMetrics{})
	html, err := reader.ReadFileAsHTML(context.Background(), "../../test/testdata/content/topic-one/file-with-properties.md")
	require.NoError(t, err)
	require.Equal(t, "<!--\ntitle: some title\n-->\n<h1>Post</h1>\n<p>With some properties</p>\n<hr>\n<h2>more: properties</h2>\n", html)
}

// verifies that an error is returned when the requested file does not exist on disk
func TestReadsFileAsHTML_MissingFile(t *testing.T) {
	reader := reading.New(nil, "", "", &MockMetrics{})
	_, err := reader.ReadFileAsHTML(context.Background(), "../../test/testdata/content/does-not-exist.md")
	require.Error(t, err)
}

// verifies that YAML-style front matter (--- ... ---) at the start of a file is stripped and does not appear in the output
func TestReadsFileAsHTML_StripsYAMLFrontMatter(t *testing.T) {
	reader := reading.New(nil, "", "", &MockMetrics{})
	html, err := reader.ReadFileAsHTML(context.Background(), "../../test/testdata/content/topic-one/yaml-front-matter.md")
	require.NoError(t, err)
	assert.Equal(t, "<h1>Content</h1>\n", html)
	assert.NotContains(t, html, "---")
	assert.NotContains(t, html, "title: Page Title")
}

// verifies that relative links pointing to files present in the index are rewritten to their indexed URI
func TestReadsFileAsHTML_ReplacesRelativeLinks(t *testing.T) {
	idx := &MockIndex{uris: map[string]string{
		"../../test/testdata/content/topic-two/README.md": "/topic-two",
	}}
	reader := reading.New(idx, "", "assets", &MockMetrics{})
	html, err := reader.ReadFileAsHTML(context.Background(), "../../test/testdata/content/topic-one/with-relative-link.md")
	require.NoError(t, err)
	assert.Equal(t, "<p><a href=\"/topic-two\">Related article</a></p>\n", html)
}

// verifies that relative links pointing to files absent from the index are left unchanged in the output
func TestReadsFileAsHTML_KeepsRelativeLinkWhenNotInIndex(t *testing.T) {
	idx := &MockIndex{uris: map[string]string{}}
	reader := reading.New(idx, "", "assets", &MockMetrics{})
	html, err := reader.ReadFileAsHTML(context.Background(), "../../test/testdata/content/topic-one/with-relative-link.md")
	require.NoError(t, err)
	assert.Equal(t, "<p><a href=\"../topic-two/README.md\">Related article</a></p>\n", html)
}

// verifies that image paths within the static content directory are rewritten to use the configured CDN URL
func TestReadsFileAsHTML_ReplacesImageLinks(t *testing.T) {
	idx := &MockIndex{uris: map[string]string{}}
	reader := reading.New(idx, "https://cdn.example.com", "assets", &MockMetrics{})
	html, err := reader.ReadFileAsHTML(context.Background(), "../../test/testdata/content/topic-one/with-image-link.md")
	require.NoError(t, err)
	assert.Equal(t, "<p><img src=\"https://cdn.example.com/assets/photo.jpg\" alt=\"Profile photo\"></p>\n", html)
}

// verifies that the full range of common markdown elements render to their expected HTML output;
// this test will catch any unintended change in rendering behaviour introduced by a library update or config change
func TestReadsFileAsHTML_RendersMarkdownElements(t *testing.T) {
	reader := reading.New(nil, "", "", &MockMetrics{})
	html, err := reader.ReadFileAsHTML(context.Background(), "../../test/testdata/content/topic-one/markdown-elements.md")
	require.NoError(t, err)

	expected := "<h1>Heading One</h1>\n" +
		"<h2>Heading Two</h2>\n" +
		"<h3>Heading Three</h3>\n" +
		"<p><strong>bold</strong> and <em>italic</em> and <code>inline code</code></p>\n" +
		"<ul>\n<li>bullet one</li>\n<li>bullet two</li>\n</ul>\n" +
		"<ol>\n<li>ordered one</li>\n<li>ordered two</li>\n</ol>\n" +
		"<blockquote>\n<p>a blockquote</p>\n</blockquote>\n" +
		"<pre><code class=\"language-go\">func hello() {}\n</code></pre>\n"

	assert.Equal(t, expected, html)
}
