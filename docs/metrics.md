# Prometheus Metrics

![alpha](assets/alpha.svg)

> v1.3 and after

## User Metrics

Each of generated EventSource, Sensor and EventBus PODs exposes an HTTP endpoint
for its metrics, which include things like how many events were generated, how
many actions were triggered, and so on. To let your Prometheus server discover
those user metrics, add following to your configuration.

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

Also please make sure your Prometheus Service Account has the permission to do
POD discovery. A sample `ClusterRole` like below needs to be added or merged,
and grant it to your Service Account.

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

#### argo_events_event_sent_total

How many events have been sent successfully.

#### argo_events_event_sent_failed_total

How many events failed to send to EventBus.

#### argo_events_event_processing_failed_total

How many events failed to process due to all the reasons, it includes
`argo_events_events_sent_failed_total`.

#### argo_events_event_processing_duration_milliseconds

Event processing duration (from getting the event to send it to EventBus) in
milliseconds.

### Sensor

#### argo_events_action_triggered_total

How many actions have been triggered successfully.

#### argo_events_action_failed_total

How many actions failed.

#### argo_events_action_duration_milliseconds

Action triggering duration.

### EventBus

For `native` NATS EventBus, check this
[link](https://github.com/nats-io/prometheus-nats-exporter) for the metrics
explanation.

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

Following metrics are considered as
[Golden Signals](https://sre.google/sre-book/monitoring-distributed-systems/#xref_monitoring_golden-signals)
of monitoring your applictions running with Argo Events.

- Latency

  - `argo_events_event_processing_duration_milliseconds`
  - `argo_events_action_duration_milliseconds`

- Traffic

  - `argo_events_event_sent_total`
  - `argo_events_action_triggered_total`

- Errors

  - `argo_events_event_processing_failed_total`
  - `argo_events_event_sent_failed_total`
  - `argo_events_action_failed_total`

- Saturation

  - `argo_events_event_service_running_total`.
  - Other Kubernetes metrics such as CPU or memory.
