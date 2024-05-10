package reading_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wamphlett/blog-server/pkg/reading"
)

type MockMetrics struct{}

func (m *MockMetrics) ParseFile(startTime time.Time)    {}
func (m *MockMetrics) ParseHeaders(startTime time.Time) {}

func TestReadsFileAsHTMLStripsProperties(t *testing.T) {
	reader := reading.New(nil, "", "", &MockMetrics{})
	html, err := reader.ReadFileAsHTML("../../test/testdata/content/topic-one/file-with-properties.md")
	require.NoError(t, err)

	require.Equal(t, "<!--\ntitle: some title\n-->\n<h1>Post</h1>\n<p>With some properties</p>\n<hr>\n<h2>more: properties</h2>\n", html)
}
