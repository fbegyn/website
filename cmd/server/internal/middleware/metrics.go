package middleware

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "handler_request_total",
			Help: "Total number of request by HTTP code",
		}, []string{"handler", "code"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "handler_request_duration",
			Help: "Durtion of the HTTP request",
		}, []string{"handler", "method"},
	)
	requestInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "handler_request_in_flight",
			Help: "Current number of request being handled",
		}, []string{"handler"},
	)
)

func init() {
	_ = prometheus.Register(requestCounter)
	_ = prometheus.Register(requestDuration)
	_ = prometheus.Register(requestInFlight)
}

func Metrics(family string, next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerDuration(
		requestDuration.MustCurryWith(prometheus.Labels{"handler": family}),
		promhttp.InstrumentHandlerCounter(
			requestCounter.MustCurryWith(prometheus.Labels{"handler": family}),
			promhttp.InstrumentHandlerInFlight(requestInFlight.With(prometheus.Labels{"handler": family}), next),
		),
	)
}
