package serving

type StatusResponse struct {
	Ready       bool  `json:"ready"`
	LastIndexed int64 `json:"lastIndexed"`
}

type CommonItemResponse struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Image       string            `json:"image"`
	URL         string            `json:"url"`
	Priority    int64             `json:"priority"`
	Slug        string            `json:"slug"`
	PublishedAt int64             `json:"publishedAt"`
	UpdatedAt   int64             `json:"updatedAt"`
	Hidden      bool              `json:"hidden"`
	Metadata    map[string]string `json:"metadata"`
}

type HtmlResponse struct {
	Html string `json:"html"`
}

type Article struct {
	CommonItemResponse
	TopicSlug string `json:"topicSlug"`
}

type GetArticleResponse struct {
	Article
	HtmlResponse
}

type Topic struct {
	CommonItemResponse
	ArticleURL            string `json:"articleUrl"`
	PublishedArticleCount int    `json:"publishedArticleCount"`
}

type OverviewResponse struct {
	HtmlResponse
}

type GetTopicResponse struct {
	Topic
	HtmlResponse
}

type ListTopicsResponse struct {
	Topics []Topic `json:"topics"`
}

type ListArticlesResponse struct {
	Articles []Article `json:"articles"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
