# Webhook Health Check

![alpha](assets/alpha.svg)

> v1.0 and after

For `webhook` or `webhook` extended event sources such as `github`, `gitlab`,
`sns`, `slack`, `Storage GRID` and `stripe`, besides the endpint configured in
the spec, an endpoint `:${port}/health` will be created by default, this is
useful for LB or Ingress configuration for the event source, where usually a
health check endpoint is required.

For example, besides `:12000/example1` and `:13000/example2`, the EventSource
object below also creates 2 extra endpoints, `:12000/health` and
`:13000/health`. A HTTP GET request to the health endpoint returns a text `OK`
with HTTP response code `200`.

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
