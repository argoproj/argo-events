package metrics

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/collectors"

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
	namespace               string
	runningEventServices    *prometheus.GaugeVec
	eventsSent              *prometheus.CounterVec
	eventsSentFailed        *prometheus.CounterVec
	eventsProcessingFailed  *prometheus.CounterVec
	eventProcessingDuration *prometheus.SummaryVec
	actionTriggered         *prometheus.CounterVec
	actionFailed            *prometheus.CounterVec
	actionDuration          *prometheus.SummaryVec
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
			Help:      "How many events failed to send to EventBus. https://argoproj.github.io/argo-events/metrics/#argo_events_events_sent_failed_total",
			ConstLabels: prometheus.Labels{
				labelNamespace: namespace,
			},
		}, []string{labelEventSourceName, labelEventName}),
		eventsProcessingFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "events_processing_failed_total",
			Help:      "Number of events failed to process, it includes argo_events_events_sent_failed_total. https://argoproj.github.io/argo-events/metrics/#argo_events_events_processing_failed_total",
			ConstLabels: prometheus.Labels{
				labelNamespace: namespace,
			},
		}, []string{labelEventSourceName, labelEventName}),
		eventProcessingDuration: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace: prefix,
			Name:      "event_processing_duration_milliseconds",
			Help:      "Summary of durations of event processing. https://argoproj.github.io/argo-events/metrics/#argo_events_event_processing_duration_milliseconds",
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
		actionDuration: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace: prefix,
			Name:      "action_duration_milliseconds",
			Help:      "Summary of durations of trigging actions. https://argoproj.github.io/argo-events/metrics/#argo_events_action_duration_milliseconds",
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
	m.eventsProcessingFailed.Collect(ch)
	m.eventProcessingDuration.Collect(ch)
	m.actionTriggered.Collect(ch)
	m.actionFailed.Collect(ch)
	m.actionDuration.Collect(ch)
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.runningEventServices.Describe(ch)
	m.eventsSent.Describe(ch)
	m.eventsSentFailed.Describe(ch)
	m.eventsProcessingFailed.Describe(ch)
	m.eventProcessingDuration.Describe(ch)
	m.actionTriggered.Describe(ch)
	m.actionFailed.Describe(ch)
	m.actionDuration.Describe(ch)
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

func (m *Metrics) EventProcessingFailed(eventSourceName, eventName string) {
	m.eventsProcessingFailed.WithLabelValues(eventSourceName, eventName).Inc()
}

func (m *Metrics) EventProcessingDuration(eventSourceName, eventName string, num float64) {
	m.eventProcessingDuration.WithLabelValues(eventSourceName, eventName).Observe(num)
}

func (m *Metrics) ActionTriggered(sensorName, triggerName string) {
	m.actionTriggered.WithLabelValues(sensorName, triggerName).Inc()
}

func (m *Metrics) ActionFailed(sensorName, triggerName string) {
	m.actionFailed.WithLabelValues(sensorName, triggerName).Inc()
}

func (m *Metrics) ActionDuration(sensorName, triggerName string, num float64) {
	m.actionDuration.WithLabelValues(sensorName, triggerName).Observe(num)
}

// Run starts a metrics server
func (m *Metrics) Run(ctx context.Context, addr string) {
	log := logging.FromContext(ctx)
	metricsRegistry := prometheus.NewRegistry()
	metricsRegistry.MustRegister(collectors.NewGoCollector(), m)
	http.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))

	log.Info("starting metrics server")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalw("failed to start metrics server", zap.Error(err))
	}
}
