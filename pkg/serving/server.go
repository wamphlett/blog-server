package serving

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"github.com/wamphlett/blog-server/pkg/telemetry"

	"github.com/wamphlett/blog-server/pkg/model"
)

// Metrics defines the metrics used by the server
type Metrics interface {
	Request(ctx context.Context, method string, status int, startTime time.Time)
}

// FileReader defines the methods required by the reader
type FileReader interface {
	ReadFileAsHTML(ctx context.Context, filepath string) (string, error)
}

// Index defines the methods required by the index
type Index interface {
	GetLastIndexedTime() time.Time
	GetAllTopics() []*model.Topic
	GetTopicByIdentifier(topicIdentidier string) *model.Topic
	GetArticleByIdentifier(topicIdentidier, identifier string) *model.Article
	GetAllArticlesForTopic(topicIdentifier string) []*model.Article
	GetRecentArticles(limit int) []*model.Article
	GetArticlesInSeries(series string) []*model.Article
}

// Server defines a new server
type Server struct {
	reader           FileReader
	index            Index
	srv              *http.Server
	router           *mux.Router
	overviewFilePath string
	metrics          Metrics
	port             int
	allowedOrigins   []string
}

// Option defines the function required to set options
type Option func(*Server)

// WithPort specifies the port to bind
func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
	}
}

// WithAllowedOrigins specifies the allowed origins to specify in the cors config
func WithAllowedOrigins(origins []string) Option {
	return func(s *Server) {
		s.allowedOrigins = origins
	}
}

// New creates a new server with the required dependencies
func New(reader FileReader, index Index, contentDir, assetDir, overviewFilePath string, metrics Metrics, opts ...Option) *Server {
	s := &Server{
		reader:           reader,
		index:            index,
		router:           mux.NewRouter(),
		overviewFilePath: filepath.Join(contentDir, overviewFilePath),
		metrics:          metrics,
		port:             3000,
		allowedOrigins:   []string{},
	}

	// apply options
	for _, opt := range opts {
		opt(s)
	}

	// serve static files
	s.router.PathPrefix(fmt.Sprintf("/%s/", assetDir)).Handler(neuter(http.FileServer(http.Dir(contentDir))))

	// set up server routes
	s.router.HandleFunc("/status", s.status)
	s.router.HandleFunc("/overview", s.getOverview)
	s.router.HandleFunc("/recent", s.getRecent)
	s.router.HandleFunc("/topics", s.listTopics)
	s.router.HandleFunc("/topics/{topic}", s.getTopic)
	s.router.HandleFunc("/topics/{topic}/articles", s.listArticles)
	s.router.HandleFunc("/topics/{topic}/articles/{article}", s.getArticle)
	s.router.Use(otelmux.Middleware(telemetry.InstrumentationName))
	s.router.Use(s.observabilityMiddleware)

	c := cors.New(cors.Options{
		AllowedOrigins: s.allowedOrigins,
	})

	s.srv = &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", s.port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      c.Handler(s.router),
	}

	return s
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(StatusResponse{
		Ready:       true,
		LastIndexed: s.index.GetLastIndexedTime().Unix(),
	})
}

func (s *Server) getOverview(w http.ResponseWriter, r *http.Request) {
	content, err := s.reader.ReadFileAsHTML(r.Context(), s.overviewFilePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to read overview file", "path", s.overviewFilePath, "error", err)
		s.internalError(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(OverviewResponse{
		HtmlResponse{content},
	})
}

func (s *Server) getRecent(w http.ResponseWriter, r *http.Request) {
	limit := 3
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		limit, _ = strconv.Atoi(limitParam)
	}

	recentArticles := s.index.GetRecentArticles(limit)
	convertedArticles := make([]Article, len(recentArticles))
	for i, article := range recentArticles {
		articleTopic := s.index.GetTopicByIdentifier(article.TopicSlug)
		if articleTopic == nil {
			slog.ErrorContext(r.Context(), "failed to find topic for article", "article", article.Slug)
			sentry.CaptureException(errors.Errorf("failed to find topic for article %s", article.Slug))
			continue
		}

		convertedArticles[i] = convertArticle(articleTopic, article)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ListArticlesResponse{
		convertedArticles,
	})
}

func (s *Server) listTopics(w http.ResponseWriter, r *http.Request) {
	topics := s.index.GetAllTopics()
	topicResponses := make([]Topic, len(topics))
	i := 0
	for _, topic := range topics {
		topicResponses[i] = convertTopic(topic, s.index.GetAllArticlesForTopic(topic.Slug))
		i++
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ListTopicsResponse{topicResponses})
}

func (s *Server) listArticles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic := s.index.GetTopicByIdentifier(vars["topic"])
	if topic == nil {
		s.notFound(w, r)
		return
	}

	topicArticles := s.index.GetAllArticlesForTopic(vars["topic"])
	articles := make([]Article, len(topicArticles))
	i := 0
	for _, article := range topicArticles {
		articles[i] = convertArticle(topic, article)
		i++
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ListArticlesResponse{articles})
}

func (s *Server) getArticle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic := s.index.GetTopicByIdentifier(vars["topic"])
	if topic == nil {
		s.notFound(w, r)
		return
	}

	article := s.index.GetArticleByIdentifier(vars["topic"], vars["article"])
	if article == nil {
		s.notFound(w, r)
		return
	}

	content, err := s.reader.ReadFileAsHTML(r.Context(), article.FilePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to read article file", "topic", vars["topic"], "article", vars["article"], "error", err)
		s.internalError(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetArticleResponse{
		convertArticle(topic, article),
		HtmlResponse{content},
		s.buildSeries(article),
	})
}

// buildSeries returns the series payload for an article, or nil if the
// article does not belong to a series
func (s *Server) buildSeries(article *model.Article) *Series {
	seriesName := article.Series()
	if seriesName == "" {
		return nil
	}

	seriesArticles := s.index.GetArticlesInSeries(seriesName)
	articles := make([]SeriesArticle, len(seriesArticles))
	for i, a := range seriesArticles {
		articles[i] = SeriesArticle{
			Title:       a.Title,
			Slug:        a.Slug,
			TopicSlug:   a.TopicSlug,
			URL:         fmt.Sprintf("/topics/%s/articles/%s", a.TopicSlug, a.Slug),
			PublishedAt: a.PublishedAt,
			Published:   a.IsPublished(),
		}
	}

	return &Series{
		Name:     seriesName,
		Articles: articles,
	}
}

func (s *Server) getTopic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic := s.index.GetTopicByIdentifier(vars["topic"])
	if topic == nil {
		s.notFound(w, r)
		return
	}

	content, err := s.reader.ReadFileAsHTML(r.Context(), topic.FilePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to read topic file", "topic", vars["topic"], "error", err)
		s.internalError(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetTopicResponse{
		convertTopic(topic, s.index.GetAllArticlesForTopic(vars["topic"])),
		HtmlResponse{content},
	})
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(ErrorResponse{"not found"})
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(ErrorResponse{"internal error"})
}

func (s *Server) observabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}

		s.metrics.Request(r.Context(), r.Method, status, start)

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}
		slog.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

func (s *Server) ListenAndServe() {
	slog.Info("server listening", "addr", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}

func (s *Server) Shutdown() {
	slog.Info("server shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
}

func buildTopicUrl(topic *model.Topic) string {
	return fmt.Sprintf("/topics/%s", topic.Slug)
}

func buildTopicArticlesUrl(topic *model.Topic) string {
	return fmt.Sprintf("%s/articles", buildTopicUrl(topic))
}

func buildArticleUrl(topic *model.Topic, article *model.Article) string {
	return fmt.Sprintf("%s/%s", buildTopicArticlesUrl(topic), article.Slug)
}

func convertTopic(topic *model.Topic, articles []*model.Article) Topic {
	publishedArticleCount := 0
	for _, article := range articles {
		if article.IsPublished() {
			publishedArticleCount++
		}
	}

	return Topic{
		CommonItemResponse{
			Title:       topic.Title,
			Description: topic.Description,
			Hidden:      topic.Hidden,
			Image:       topic.Image,
			URL:         buildTopicUrl(topic),
			Priority:    topic.Priority,
			Slug:        topic.Slug,
			PublishedAt: topic.PublishedAt,
			UpdatedAt:   topic.UpdatedAt,
			Metadata:    topic.Metadata,
		},
		buildTopicArticlesUrl(topic),
		publishedArticleCount,
	}
}

func convertArticle(topic *model.Topic, article *model.Article) Article {
	return Article{
		CommonItemResponse{
			Title:       article.Title,
			Description: article.Description,
			Hidden:      article.Hidden,
			Image:       article.Image,
			URL:         buildArticleUrl(topic, article),
			Priority:    article.Priority,
			Slug:        article.Slug,
			PublishedAt: article.PublishedAt,
			UpdatedAt:   article.UpdatedAt,
			Metadata:    article.Metadata,
		},
		topic.Slug,
	}
}

func neuter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

