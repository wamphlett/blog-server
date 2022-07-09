package serving

type CommonItemResponse struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
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
