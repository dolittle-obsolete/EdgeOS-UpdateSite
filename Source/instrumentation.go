package main

import (
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type instrumentedSwupdServer struct {
	handler    http.Handler
	inFlight   prometheus.Gauge
	total      *prometheus.CounterVec
	bytes      *prometheus.CounterVec
	duration   prometheus.ObserverVec
	sizes      prometheus.ObserverVec
	throughput prometheus.ObserverVec
	images     *prometheus.CounterVec
	updates    *prometheus.CounterVec
}

func (server *instrumentedSwupdServer) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	server.inFlight.Inc()
	defer server.inFlight.Dec()

	capturer := &responseCapturer{
		original: response,
	}
	start := time.Now()

	server.handler.ServeHTTP(capturer, request)

	duration := time.Since(start).Seconds()
	code, method := getStatusAndMethodLabels(capturer.statusCode, request.Method)

	server.total.WithLabelValues(code, method).Inc()
	server.bytes.WithLabelValues(code, method).Add(float64(capturer.written))
	server.duration.WithLabelValues(code, method).Observe(duration)
	server.sizes.WithLabelValues(code, method).Observe(float64(capturer.written))
	server.throughput.WithLabelValues(code, method).Observe(float64(capturer.written) / duration)

	server.incrementDownloadsCount(code, request)
}

var (
	imageRegex  = regexp.MustCompile(`^/images/([^/]+)/(.+)`)
	updateRegex = regexp.MustCompile(`^/update/([^/]+)/([0-9]+)/`)
)

func (server *instrumentedSwupdServer) incrementDownloadsCount(code string, request *http.Request) {
	upath := path.Clean("/" + request.URL.Path)
	if match := imageRegex.FindStringSubmatch(upath); match != nil {
		release := match[1]
		image := match[2]
		server.images.WithLabelValues(code, release, image).Inc()
	} else if match := updateRegex.FindStringSubmatch(upath); match != nil {
		release := match[1]
		version := match[2]
		server.updates.WithLabelValues(code, release, version).Inc()
	}
}

func instrumentSwupdServer(handler http.Handler) http.Handler {
	return &instrumentedSwupdServer{
		handler: handler,
		inFlight: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "swupd_http_requests_inflight",
			Help: "Current number of HTTP requests in flight for the SWUPD server",
		}),
		total: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "swupd_http_requests_total",
			Help: "Total number of HTTP requests to the SWUPD server",
		}, []string{"code", "method"}),
		bytes: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "swupd_http_response_bytes_total",
			Help: "Total number of HTTP response bytes from the SWUPD server",
		}, []string{"code", "method"}),
		duration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "swupd_http_request_duration_seconds",
			Help:    "Duration of HTTP requests to the SWUPD server",
			Buckets: []float64{0.1, 1, 10, 60, 600},
		}, []string{"code", "method"}),
		sizes: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "swupd_http_response_size_bytes",
			Help:    "Size of HTTP responses from the SWUPD server",
			Buckets: []float64{100 * 1024, 1024 * 1024, 10 * 1024 * 1024, 100 * 1024 * 1024, 1024 * 1024 * 1024},
		}, []string{"code", "method"}),
		throughput: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "swupd_http_response_throughput",
			Help:    "Throughput of HTTP responses from the SWUPD server",
			Buckets: []float64{1024, 100 * 1024, 1024 * 1024, 10 * 1024 * 1024, 100 * 1024 * 1024},
		}, []string{"code", "method"}),
		images: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "swupd_image_downloads_total",
			Help: "Total number image downloads from the SWUPD server",
		}, []string{"code", "release", "image"}),
		updates: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "swupd_update_downloads_total",
			Help: "Total update files downloaded from the SWUPD server",
		}, []string{"code", "release", "version"}),
	}
}

type responseCapturer struct {
	original   http.ResponseWriter
	statusCode int
	written    int
}

func (c *responseCapturer) Header() http.Header {
	return c.original.Header()
}

func (c *responseCapturer) WriteHeader(statusCode int) {
	c.statusCode = statusCode
	c.original.WriteHeader(statusCode)
}

func (c *responseCapturer) Write(data []byte) (int, error) {
	if c.statusCode == 0 {
		c.statusCode = http.StatusOK
	}
	written, err := c.original.Write(data)
	c.written += written
	return written, err
}

func getStatusAndMethodLabels(statusCode int, method string) (string, string) {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return strconv.Itoa(statusCode), strings.ToLower(method)
}
