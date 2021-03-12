# Calender EventSource Catch Up

Catch-up feature allow Calender eventsources to execute the missed schedules
from last run.

## Enable Catch-up for Calendar EventSource

User can configure catch up on each events in eventsource.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: calendar
spec:
  template:
    serviceAccountName: configmap-sa # assign a service account with read, write permissions on configmaps
  calendar:
    example-with-catch-up:
      # Catchup the missed events from last Event timestamp. last event will be persisted in configmap.
      schedule: "* * * * *"
      persistence:
        catchup:
          enabled: true # Check missed schedules from last persisted event time on every start
          maxDuration: 5m # maximum amount of duration go back for the catch-up
        configMap: # Configmap for persist the last successful event timestamp
          createIfNotExist: true
          name: test-configmap
```

Last calender event persisted in configured configmap. Same configmap can be
used by multiple events configuration.

```yaml
data:
  calendar.example-with-catch-up:
    '{"eventTime":"2020-10-19 22:50:00.0003192 +0000 UTC m=+683.567066901"}'
```

### Service Account

To make Calendar EventSource catch-up work, a Service Account with proper RBAC
settings needs to be provided.

If the configMap is not existing, and `createIfNotExist: true` is set, a Service
Account bound with following `Role` is required.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: example-configmap-access-role
rules:
  - apiGroups:
      - ""
    resources:
      - configmaps
    verbs:
      - get
      - create
      - update
```

If the configmap is already existing, `create` can be removed from the `verbs`
list.

## Disable the catchup

Set `false` to catchup-->enabled element

```yaml
catchup:
  enabled: false
```
