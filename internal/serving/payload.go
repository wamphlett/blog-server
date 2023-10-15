package serving

type CommonItemResponse struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Image       string            `json:"image"`
	URL         string            `json:"url"`
	Priority    int64             `json:"priority"`
	Slug        string            `json:"slug"`
	Metadata    map[string]string `json:"metadata"`
}

type HtmlResponse struct {
	Html string `json:"html"`
}

type Article struct {
	CommonItemResponse
}

type GetArticleResponse struct {
	Article
	HtmlResponse
}

type Topic struct {
	CommonItemResponse
	ArticleURL string `json:"articleUrl"`
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
