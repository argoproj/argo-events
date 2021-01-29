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
	namespace       string
	sensorName      string
	actionTriggered *prometheus.CounterVec
	actionFailed    *prometheus.CounterVec
}

// NewMetrics returns a Metrics instance
func NewMetrics(namespace, sensorName string) *Metrics {
	return &Metrics{
		namespace:  namespace,
		sensorName: sensorName,
		actionTriggered: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "action_triggered_total",
			Help:      "How many actions have been triggered successfully",
			ConstLabels: prometheus.Labels{
				"namespace":   namespace,
				"sensor_name": sensorName,
			},
		}, []string{"trigger_name"}),
		actionFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "action_failed_total",
			Help:      "How many actions failed",
			ConstLabels: prometheus.Labels{
				"namespace":   namespace,
				"sensor_name": sensorName,
			},
		}, []string{"trigger_name"}),
	}
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.actionTriggered.Collect(ch)
	m.actionFailed.Collect(ch)
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.actionTriggered.Describe(ch)
	m.actionFailed.Describe(ch)
}

func (m *Metrics) ActionTriggered(triggerName string) {
	m.actionTriggered.WithLabelValues(triggerName).Inc()
}

func (m *Metrics) ActionFailed(triggerName string) {
	m.actionFailed.WithLabelValues(triggerName).Inc()
}

// Run starts a metrics server
func (m *Metrics) Run(ctx context.Context, addr string) {
	log := logging.FromContext(ctx)
	metricsRegistry := prometheus.NewRegistry()
	metricsRegistry.MustRegister(m)
	http.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))
	log.Info("starting sensor metrics server")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalw("failed to start sensor metrics server", zap.Error(err))
	}
}
