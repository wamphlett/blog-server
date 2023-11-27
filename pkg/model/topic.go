package model

// Topic defines a topic entry
type Topic struct {
	Title       string
	Description string
	Image       string

	Slug   string
	URI    string
	Hidden bool

	FilePath string

	Priority    int64
	PublishedAt int64
	UpdatedAt   int64
	Metadata    map[string]string
}
