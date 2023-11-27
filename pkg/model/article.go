package model

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
	Priority    int64
	Metadata    map[string]string
}
