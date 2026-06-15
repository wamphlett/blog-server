package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Client holds all Prometheus metrics for the blog server.
type Client struct {
	httpRequestDuration *prometheus.HistogramVec
	httpRequestsTotal   *prometheus.CounterVec

	parseFileDuration prometheus.Histogram
	parseFileTotal    prometheus.Counter

	parseHeadersDuration prometheus.Histogram
	parseHeadersTotal    prometheus.Counter

	contentUpdateDuration prometheus.Histogram
	contentUpdatesTotal   prometheus.Counter
	contentLastUpdated    prometheus.Gauge

	indexDuration prometheus.Histogram
	indexTotal    prometheus.Counter
	topicsTotal   prometheus.Gauge
	articlesTotal prometheus.Gauge
}

// New creates a new metrics client and registers all Prometheus metrics.
func New() *Client {
	return &Client{
		httpRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "status_code"}),
		httpRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		}, []string{"method", "path", "status_code"}),

		parseFileDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "blog_server_parse_file_duration_seconds",
			Help:    "Duration of markdown file parse operations",
			Buckets: prometheus.DefBuckets,
		}),
		parseFileTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "blog_server_parse_file_total",
			Help: "Total number of markdown files parsed",
		}),

		parseHeadersDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "blog_server_parse_headers_duration_seconds",
			Help:    "Duration of content header parse operations",
			Buckets: prometheus.DefBuckets,
		}),
		parseHeadersTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "blog_server_parse_headers_total",
			Help: "Total number of content header parse operations",
		}),

		contentUpdateDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "blog_server_content_update_duration_seconds",
			Help:    "Duration of content update operations",
			Buckets: prometheus.DefBuckets,
		}),
		contentUpdatesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "blog_server_content_updates_total",
			Help: "Total number of content update cycles",
		}),
		contentLastUpdated: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "blog_server_content_last_updated_timestamp_seconds",
			Help: "Unix timestamp of the most recent content update",
		}),

		indexDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "blog_server_index_duration_seconds",
			Help:    "Duration of content indexing operations",
			Buckets: prometheus.DefBuckets,
		}),
		indexTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "blog_server_index_total",
			Help: "Total number of indexing operations",
		}),
		topicsTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "blog_server_topics_total",
			Help: "Number of topics currently indexed",
		}),
		articlesTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "blog_server_articles_total",
			Help: "Number of articles currently indexed",
		}),
	}
}
