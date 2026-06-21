package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/wamphlett/blog-server/pkg/model"
)

// verifies that IsPublished returns true only when the article has a past publish date and is not hidden
func TestIsPublished(t *testing.T) {
	past := time.Now().Add(-time.Hour).Unix()
	future := time.Now().Add(time.Hour).Unix()

	tests := []struct {
		name      string
		article   model.Article
		published bool
	}{
		{
			name:      "published article with past date",
			article:   model.Article{PublishedAt: past, Hidden: false},
			published: true,
		},
		{
			name:      "future article not yet live",
			article:   model.Article{PublishedAt: future, Hidden: false},
			published: false,
		},
		{
			name:      "hidden article is not published",
			article:   model.Article{PublishedAt: past, Hidden: true},
			published: false,
		},
		{
			name:      "article with zero published date",
			article:   model.Article{PublishedAt: 0, Hidden: false},
			published: false,
		},
		{
			name:      "hidden future article",
			article:   model.Article{PublishedAt: future, Hidden: true},
			published: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.published, tt.article.IsPublished())
		})
	}
}
