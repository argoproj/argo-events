# Calendar Gateway & Sensor

Intervals or cron schedules act as event sources for calendar gateway.

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)

## Example event sources definition
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: calendar-gateway-configmap
data:
  interval: |-
    interval: 10s # event is generated after every 10 seconds
  schedule: |-
    schedule: 30 * * * *  # event is generated after 30 min past every hour
```

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/calendar-gateway-configmap.yaml
```

## Install gateway
1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/calendar.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway calendar-gateway
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/calendar.yaml
```
