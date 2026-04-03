# Prometheus Metrics

![alpha](assets/alpha.svg)

> v1.3 and after

## User Metrics

Each of generated EventSource, Sensor and EventBus PODs exposes an HTTP endpoint
for its metrics, which include things like how many events were generated, how
many actions were triggered, and so on. To let your Prometheus server discover
those user metrics, add the following to your configuration.

```txt
    - job_name: 'argo-events'
      kubernetes_sd_configs:
      - role: pod
        selectors:
        - role: pod
          label: 'controller in (eventsource-controller,sensor-controller,eventbus-controller)'
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_eventbus_name, __meta_kubernetes_pod_label_controller]
        action: replace
        regex: (.+);eventbus-controller
        replacement: $1
        target_label: 'eventbus_name'
      - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_controller]
        action: replace
        regex: (.+);eventbus-controller
        replacement: $1
        target_label: 'namespace'
      - source_labels: [__address__, __meta_kubernetes_pod_label_controller]
        action: drop
        regex: (.+):(\d222);eventbus-controller
```

Also, please make sure your Prometheus Service Account has the permission to do
POD discovery. A sample `ClusterRole` like the one below needs to be added or merged
and granted to your Service Account.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pod-discovery
rules:
  - apiGroups: [""]
    resources:
      - pods
    verbs: ["get", "list", "watch"]
```

### EventSource

#### argo_events_event_service_running_total

How many configured events in the EventSource object are actively running.

#### argo_events_events_sent_total

How many events have been sent successfully.

#### argo_events_events_sent_failed_total

How many events failed to send to EventBus.

#### argo_events_events_processing_failed_total

How many events failed to process for any reason. This includes
`argo_events_events_sent_failed_total`.

#### argo_events_event_processing_duration_milliseconds

Event processing duration (from receiving the event to sending it to EventBus) in
milliseconds.

### Sensor

#### argo_events_action_triggered_total

How many actions have been triggered successfully.

#### argo_events_action_failed_total

How many actions failed.

#### argo_events_action_retries_failed_total

How many actions failed after retries have been exhausted.
This is also incremented if no `retryStrategy` is specified.

#### argo_events_action_duration_milliseconds

Action triggering duration.

### EventBus

For the `native` NATS EventBus, check this
[link](https://github.com/nats-io/prometheus-nats-exporter) for metrics
explanations.

## Controller Metrics

If you are interested in Argo Events controller metrics, add following to your
Prometheus configuration.

```txt
    - job_name: 'argo-events-controllers'
      kubernetes_sd_configs:
      - role: pod
        selectors:
        - role: pod
          label: 'app in (eventsource-controller,sensor-controller,eventbus-controller)'
      relabel_configs:
      - source_labels: [__address__, __meta_kubernetes_pod_label_app]
        action: replace
        regex: (.+);(eventsource-controller|sensor-controller|eventbus-controller)
        replacement: $1:7777
        target_label: '__address__'
      - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app]
        action: replace
        regex: (.+);(eventsource-controller|sensor-controller|eventbus-controller)
        replacement: $1
        target_label: 'namespace'
```

## Golden Signals

The following metrics are considered
[Golden Signals](https://sre.google/sre-book/monitoring-distributed-systems/#xref_monitoring_golden-signals)
for monitoring your applications running with Argo Events.

- Latency

  - `argo_events_event_processing_duration_milliseconds`
  - `argo_events_action_duration_milliseconds`

- Traffic

  - `argo_events_events_sent_total`
  - `argo_events_action_triggered_total`

- Errors

  - `argo_events_events_processing_failed_total`
  - `argo_events_events_sent_failed_total`
  - `argo_events_action_failed_total`
  - `argo_events_action_retries_failed_total`

- Saturation

  - `argo_events_event_service_running_total`.
  - Other Kubernetes metrics such as CPU or memory.

## OpenTelemetry Distributed Tracing

Argo Events supports OpenTelemetry distributed tracing for end-to-end
visibility across the event pipeline (EventSource -> EventBus -> Sensor -> Trigger).

Tracing is **opt-in** and controlled entirely via standard OpenTelemetry
environment variables. When not configured, there is zero performance impact.

### Configuration

Add the following environment variables to your EventSource and Sensor pod
templates to enable tracing:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: webhook
spec:
  template:
    container:
      env:
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "http://jaeger-collector.observability:4317"
        - name: OTEL_TRACES_EXPORTER
          value: "otlp"
  webhook:
    example:
      port: "12000"
      endpoint: /example
      method: POST
```

The same environment variables should be added to the Sensor spec:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: webhook
spec:
  template:
    container:
      env:
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "http://jaeger-collector.observability:4317"
        - name: OTEL_TRACES_EXPORTER
          value: "otlp"
  # ... dependencies and triggers
```

### Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint (e.g., `http://jaeger:4317`) | unset (tracing disabled) |
| `OTEL_TRACES_EXPORTER` | Set to `none` to explicitly disable tracing | `otlp` |
| `OTEL_SERVICE_NAME` | Override the default service name | `argo-events-eventsource` or `argo-events-sensor` |

### Trace Spans

When tracing is enabled, the following spans are created:

- **`eventsource.publish`** - Created when an event source publishes an event to
  the EventBus. Attributes: `eventsource.name`, `eventsource.type`, `event.name`,
  `event.id`.

- **`sensor.trigger`** - Created when a sensor executes a trigger. Attributes:
  `sensor.name`, `trigger.name`, `dependencies`, `event.ids`.

### CloudEvents Trace Context Propagation

Argo Events preserves
[W3C Trace Context](https://github.com/cloudevents/spec/blob/main/cloudevents/extensions/distributed-tracing.md)
from incoming CloudEvents. When an external system sends a CloudEvent with
`traceparent`/`tracestate` headers (or extensions), the trace context is:

1. Preserved in the event as it flows through the EventBus.
2. Used as the parent span for `eventsource.publish`.
3. Propagated to `sensor.trigger`, creating a connected trace across the pipeline.

This enables end-to-end distributed traces from an external event producer
through Argo Events to the triggered action.
