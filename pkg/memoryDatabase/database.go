package memorydatabase

import (
	"github.com/wamphlett/blog-server/pkg/model"
)

type Database struct {
	topics   map[string]*model.Topic
	articles map[string]map[string]*model.Article
}

func New() *Database {
	return &Database{
		topics:   map[string]*model.Topic{},
		articles: map[string]map[string]*model.Article{},
	}
}

func (d *Database) StoreTopic(topic *model.Topic) {
	d.topics[topic.Slug] = topic
}

func (d *Database) StoreArticle(article *model.Article) {
	if _, ok := d.articles[article.TopicSlug]; !ok {
		d.articles[article.TopicSlug] = map[string]*model.Article{}
	}
	d.articles[article.TopicSlug][article.Slug] = article
}

func (d *Database) GetAllTopics() []*model.Topic {
	topics := make([]*model.Topic, 0, len(d.topics))
	for _, topic := range d.topics {
		topics = append(topics, topic)
	}
	return topics
}

func (d *Database) GetAllArticles() []*model.Article {
	articles := []*model.Article{}
	for _, topicArticles := range d.articles {
		for _, article := range topicArticles {
			articles = append(articles, article)
		}
	}
	return articles
}

func (d *Database) GetAllArticlesForTopic(topicSlug string) []*model.Article {
	if _, ok := d.articles[topicSlug]; !ok {
		return []*model.Article{}
	}

	articles := make([]*model.Article, 0, len(d.articles[topicSlug]))
	for _, article := range d.articles[topicSlug] {
		articles = append(articles, article)
	}
	return articles
}
