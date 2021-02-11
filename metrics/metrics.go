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

	labelNamespace       = "namespace"
	labelEventSourceName = "eventsource_name"
	labelEventName       = "event_name"
	labelSensorName      = "sensor_name"
	labelTriggerName     = "trigger_name"
)

// Metrics represents EventSource metrics information
type Metrics struct {
	namespace            string
	runningEventServices *prometheus.GaugeVec
	eventsSent           *prometheus.CounterVec
	eventsSentFailed     *prometheus.CounterVec
	actionTriggered      *prometheus.CounterVec
	actionFailed         *prometheus.CounterVec
}

// NewMetrics returns a Metrics instance
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		namespace: namespace,
		runningEventServices: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "event_service_running_total",
			Help:      "How many configured events in the EventSource object are actively running. https://argoproj.github.io/argo-events/metrics/#argo_events_event_service_running_total",
			ConstLabels: prometheus.Labels{
				labelNamespace: namespace,
			},
		}, []string{labelEventSourceName}),
		eventsSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "events_sent_total",
			Help:      "How many events have been sent successfully. https://argoproj.github.io/argo-events/metrics/#argo_events_events_sent_total",
			ConstLabels: prometheus.Labels{
				labelNamespace: namespace,
			},
		}, []string{labelEventSourceName, labelEventName}),
		eventsSentFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "events_sent_failed_total",
			Help:      "How many events failed to send. https://argoproj.github.io/argo-events/metrics/#argo_events_events_sent_failed_total",
			ConstLabels: prometheus.Labels{
				labelNamespace: namespace,
			},
		}, []string{labelEventSourceName, labelEventName}),
		actionTriggered: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "action_triggered_total",
			Help:      "How many actions have been triggered successfully. https://argoproj.github.io/argo-events/metrics/#argo_events_action_triggered_total",
			ConstLabels: prometheus.Labels{
				labelNamespace: namespace,
			},
		}, []string{labelSensorName, labelTriggerName}),
		actionFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "action_failed_total",
			Help:      "How many actions failed. https://argoproj.github.io/argo-events/metrics/#argo_events_action_failed_total",
			ConstLabels: prometheus.Labels{
				labelNamespace: namespace,
			},
		}, []string{labelSensorName, labelTriggerName}),
	}
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.runningEventServices.Collect(ch)
	m.eventsSent.Collect(ch)
	m.eventsSentFailed.Collect(ch)
	m.actionTriggered.Collect(ch)
	m.actionFailed.Collect(ch)
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.runningEventServices.Describe(ch)
	m.eventsSent.Describe(ch)
	m.eventsSentFailed.Describe(ch)
	m.actionTriggered.Describe(ch)
	m.actionFailed.Describe(ch)
}

func (m *Metrics) IncRunningServices(eventSourceName string) {
	m.runningEventServices.WithLabelValues(eventSourceName).Inc()
}

func (m *Metrics) DecRunningServices(eventSourceName string) {
	m.runningEventServices.WithLabelValues(eventSourceName).Dec()
}

func (m *Metrics) EventSent(eventSourceName, eventName string) {
	m.eventsSent.WithLabelValues(eventSourceName, eventName).Inc()
}

func (m *Metrics) EventSentFailed(eventSourceName, eventName string) {
	m.eventsSentFailed.WithLabelValues(eventSourceName, eventName).Inc()
}

func (m *Metrics) ActionTriggered(sensorName, triggerName string) {
	m.actionTriggered.WithLabelValues(sensorName, triggerName).Inc()
}

func (m *Metrics) ActionFailed(sensorName, triggerName string) {
	m.actionFailed.WithLabelValues(sensorName, triggerName).Inc()
}

// Run starts a metrics server
func (m *Metrics) Run(ctx context.Context, addr string) {
	log := logging.FromContext(ctx)
	metricsRegistry := prometheus.NewRegistry()
	metricsRegistry.MustRegister(m)
	http.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))
	log.Info("starting metrics server")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalw("failed to start metrics server", zap.Error(err))
	}
}
