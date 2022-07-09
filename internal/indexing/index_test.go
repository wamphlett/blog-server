package indexing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGettingArticlesByPath(t *testing.T) {
	article1 := &Article{
		Path: "article-1",
	}
	article2 := &Article{
		Path: "subdir/article-2",
	}

	tt := map[string]struct {
		path            string
		expectedArticle *Article
	}{
		"valid path 1":                       {"/topic-path/article-1", article1},
		"valid path 2":                       {"/topic-path/subdir/article-2", article2},
		"valid path 1 without leading slash": {"topic-path/article-1", article1},
		"invalid path":                       {"invalid-path", nil},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			indexer := &Index{
				topics: []*Topic{{
					Path:     "topic-path",
					Articles: []*Article{article1, article2},
				}},
			}
			article, err := indexer.GetArticleByPath(tc.path)
			if tc.expectedArticle == nil {
				assert.Error(t, err, "expected error when path does not match any articles")
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.expectedArticle, article)
		})
	}
}
