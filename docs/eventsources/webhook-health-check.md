# Webhook Health Check

For `webhook` or `webhook` extended event sources such as `github`, `gitlab`,
`sns`, `slack`, `Storage GRID` and `stripe`, besides the endpoint configured in
the spec, an extra endpoint `:${port}/health` will also be created, this is
useful for LB or Ingress configuration for the event source, where usually a
health check endpoint is required.

For example, the following EventSource object will have 4 endpoints created,
`:12000/example1`, `:12000/health`, `:13000/example2` and `:13000/health`. An
HTTP GET request to the health endpoint returns a text `OK` with HTTP response
code `200`.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: webhook
spec:
  webhook:
    example:
      port: "12000"
      endpoint: /example1
      method: POST
    example-foo:
      port: "13000"
      endpoint: /example2
      method: POST
```
