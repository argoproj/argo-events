# EventSource Services

Some of the EventSources (`webhook`, `github`, `gitlab`, `sns`, `slack`,
`Storage GRID` and `stripe`) start an HTTP service to receive the events, for
your convenience, there is a field named `service` within EventSource spec can
help you create a `ClusterIP` service for testing.

For example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: webhook
spec:
  service:
    ports:
      - port: 12000
        targetPort: 12000
  webhook:
    example:
      port: "12000"
      endpoint: /example
      method: POST
```

However, the generated service is **ONLY** for testing purpose, if you want to
expose the endpoint for external access, please manage it by using native K8s
objects (i.e. a Load Balancer type Service, or an Ingress), and remove `service`
field from the EventSource object.

You can refer to [webhook heath check](../webhook-health-check.md) if you need a
health check endpoint for LB Service or Ingress configuration.
