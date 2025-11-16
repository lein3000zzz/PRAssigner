package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.DefaultRegisterer

	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests by path/method/code.",
		},
		[]string{"path", "method", "code"},
	)

	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests by path/method/code.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "code"},
	)

	prEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_events_total",
			Help: "PR events by operation and result with error label.",
		},
		[]string{"op", "result", "error"},
	)

	prDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pr_operation_duration_seconds",
			Help:    "Duration of PR repo operations by op and result.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"op", "result"},
	)

	openPRs = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "open_prs",
			Help: "Approximate number of open PRs maintained by app flow.",
		},
	)
)

func GinMiddleware(c *gin.Context) {
	start := time.Now()
	c.Next()
	code := strconv.Itoa(c.Writer.Status())
	path := c.FullPath()

	// на случаи 404, потому что иначе не записывалось бы
	if path == "" {
		path = c.Request.URL.Path
	}

	if path == "/metrics" || strings.HasPrefix(path, "/debug/pprof/") {
		return
	}

	method := c.Request.Method

	httpRequests.WithLabelValues(path, method, code).Inc()
	httpDuration.WithLabelValues(path, method, code).Observe(time.Since(start).Seconds())
}

func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func ObservePROp(op string, start time.Time, err error) {
	result := "success"
	errLabel := ""
	if err != nil {
		result = "error"
		errLabel = err.Error()
	}
	prEvents.WithLabelValues(op, result, errLabel).Inc()
	prDuration.WithLabelValues(op, result).Observe(time.Since(start).Seconds())
}

func AddOpenPR(delta float64) {
	openPRs.Add(delta)
}

func init() {
	collectors := []prometheus.Collector{
		httpRequests,
		httpDuration,
		prEvents,
		prDuration,
		openPRs,
	}

	for _, c := range collectors {
		_ = registry.Register(c)
	}
}
