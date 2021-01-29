package metrics

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common/logging"
)

const (
	prefix = "argo_events"
)

// Metrics represents EventSource metrics information
type Metrics struct {
	namespace            string
	eventSourceName      string
	runningEventServices prometheus.Gauge
	eventsSent           *prometheus.CounterVec
	eventsSentFailed     *prometheus.CounterVec
}

// NewMetrics returns a Metrics instance
func NewMetrics(namespace, eventSourceName string) *Metrics {
	return &Metrics{
		namespace:       namespace,
		eventSourceName: eventSourceName,
		runningEventServices: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "event_service_running_total",
			Help:      "How many configured events in the EventSource object are actively running",
			ConstLabels: prometheus.Labels{
				"namespace":        namespace,
				"eventsource_name": eventSourceName,
			},
		}),
		eventsSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "events_sent_total",
			Help:      "How many events have been sent successfully",
			ConstLabels: prometheus.Labels{
				"namespace":        namespace,
				"eventsource_name": eventSourceName,
			},
		}, []string{"event_name"}),
		eventsSentFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "events_sent_failed_total",
			Help:      "How many events failed to send",
			ConstLabels: prometheus.Labels{
				"namespace":        namespace,
				"eventsource_name": eventSourceName,
			},
		}, []string{"event_name"}),
	}
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.runningEventServices.Collect(ch)
	m.eventsSent.Collect(ch)
	m.eventsSentFailed.Collect(ch)
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.runningEventServices.Describe(ch)
	m.eventsSent.Describe(ch)
	m.eventsSentFailed.Describe(ch)
}

func (m *Metrics) IncRunningServices() {
	m.runningEventServices.Inc()
}

func (m *Metrics) DecRunningServices() {
	m.runningEventServices.Dec()
}

func (m *Metrics) EventSent(eventName string) {
	m.eventsSent.WithLabelValues(eventName).Inc()
}

func (m *Metrics) EventSentFailed(eventName string) {
	m.eventsSentFailed.WithLabelValues(eventName).Inc()
}

// Run starts a metrics server
func (m *Metrics) Run(ctx context.Context, addr string) {
	log := logging.FromContext(ctx)
	metricsRegistry := prometheus.NewRegistry()
	metricsRegistry.MustRegister(m)
	http.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))
	log.Info("starting eventsource metrics server")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalw("failed to start eventsource metrics server", zap.Error(err))
	}
}
