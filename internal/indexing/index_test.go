package indexing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGettingArticlesByURI(t *testing.T) {
	article1 := &Article{
		Slug: "article-1",
	}
	article2 := &Article{
		Slug: "subdir/article-2",
	}

	tt := map[string]struct {
		uri             string
		expectedArticle *Article
	}{
		"valid URI 1":                       {"/topic-path/article-1", article1},
		"valid URI 2":                       {"/topic-path/subdir/article-2", article2},
		"valid URI 1 without leading slash": {"topic-path/article-1", article1},
		"invalid URI":                       {"invalid-path", nil},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			indexer := &Index{
				topics: []*Topic{{
					Slug:     "topic-path",
					Articles: []*Article{article1, article2},
				}},
			}
			article, err := indexer.GetArticleByURI(tc.uri)
			if tc.expectedArticle == nil {
				assert.Error(t, err, "expected error when path does not match any articles")
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.expectedArticle, article)
		})
	}
}
