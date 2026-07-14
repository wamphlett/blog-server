package model

import "time"

// Article defines the information held about an article
type Article struct {
	Title       string
	Description string
	Image       string

	Slug      string
	TopicSlug string
	URI       string
	Hidden    bool

	FilePath string

	PublishedAt int64
	UpdatedAt   int64
	// CreatedAt is the timestamp of the commit that first added this file to
	// the content repository, used to order undated articles when no
	// published date is available. It is 0 if the content isn't backed by a
	// git repository or the timestamp couldn't be determined.
	CreatedAt int64
	Priority  int64
	Metadata  map[string]string
}

func (a *Article) IsPublished() bool {
	return a.PublishedAt > 0 && !a.Hidden && a.PublishedAt < time.Now().Unix()
}

// Series returns the name of the series this article belongs to,
// or an empty string if the article is not part of a series
func (a *Article) Series() string {
	return a.Metadata["series"]
}
