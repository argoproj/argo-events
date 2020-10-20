# Calender EventSource - Catchup 
Catchup feature allow Calender eventsources to execute the missed schedules from last run.
## Enable Catchup forEventSource Definition
User can configure catchup on each events in eventsource.

```yaml
    example-with-catchup:
    # Catchup the missed events from last Event timestamp. last event will be persisted in configmap.
      schedule: "* * * * *"
      persistence:
        catchup:
          enabled: true   # Check missed schedules from last persisted event time on Every start  
          maxDuration: 5m # maximum amount of duration go back for the catchup
        configMap:        # Configmap for persist the last successful event timestamp
          createIfNotExist: true
          name: test-configmap
```
Last calender event persisted in configured configmap. Multiple event can use the same configmap to persist the events.
```yaml
data:
  calendar.example-with-catchup: '{"eventTime":"2020-10-19 22:50:00.0003192 +0000 UTC m=+683.567066901"}'
```

## Disable the catchup
Set `false` to catchup-->enabled element

```yaml
catchup:
          enabled: false 
```