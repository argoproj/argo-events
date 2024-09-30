# Log

Log trigger is for debugging - it just logs events it receives as JSON:

```json
{
  "level": "info",
  "ts": 1604783266.973979,
  "logger": "argo-events.sensor",
  "caller": "log/log.go:35",
  "msg": "{\"eventTime\":\"2020-11-07 21:07:46.9658533 +0000 UTC m=+20468.986115001\"}",
  "sensorName": "log",
  "triggerName": "log-trigger",
  "dependencyName": "test-dep",
  "eventContext": "{\"id\":\"37363664356662642d616364322d343563332d396362622d353037653361343637393237\",\"source\":\"calendar\",\"specversion\":\"1.0\",\"type\":\"calendar\",\"datacontenttype\":\"application/json\",\"subject\":\"example-with-interval\",\"time\":\"2020-11-07T21:07:46Z\"}"
}
```

## Specification

The specification is available [here](../../APIs.md#argoproj.io/v1alpha1.LogTrigger).

## Parameterization

No parameterization is supported.
