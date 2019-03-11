<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/calendar.png?raw=true" alt="Calendar"/>
</p>

<br/>

# Calendar

Calendar gateway can be used when you to schedule K8s resources on an interval
or cron schedule.

## How to define an entry in confimap?
You can construct an entry in configmap using following fields,

```go
// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
Schedule string `json:"schedule"`

// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
Interval string `json:"interval"`

// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
// RDATE defines the list of DATE-TIME values for recurring events.
// EXDATE defines the list of DATE-TIME exceptions for recurring events.
// the combination of these rules and dates combine to form a set of date times.
// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
// +Optional
Recurrence []string `json:"recurrence,omitempty"`

// Timezone in which to run the schedule
// +optional
Timezone string `json:"timezone,omitempty"`

// UserPayload will be sent to sensor as extra data once the event is triggered
// +optional
UserPayload string `json:"userPayload,omitempty"`
```

### Example
In the following gateway configmap contains three event source configurations,

The `interval` configuration will basically generate an event after every 55 seconds.

The `schedule` configuration will generate an event every 30 minutes past an hour

The `withdata` configuration will generate an event every 30 minutes past an hour but the time zone is "America/New_York"
and along with event time, event payload will also contain a string  "{\r\n\"hello\": \"world\"\r\n}"

**What is the use of `userPayload`?**
Many times having only the event time in not sufficient but rather have an accompanying user defined data.
This user defined data is `userPayload`. You can basically place any string here and gateway will put that in event payload.  
 
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: calendar-gateway-configmap
data:
  interval: |-
    interval: 55s
  schedule: |-
    schedule: 30 * * * *
  withdata: |-
    schedule: 30 * * * *
    userPayload: "{\r\n\"hello\": \"world\"\r\n}"
    timezone: "America/New_York"
```

## Setup
**1. Install Gateway Configmap**

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/calendar-gateway-configmap.yaml
```

**2. Install Gateway**

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/calendar.yaml
```

Make sure the gateway pod is created.

**3. Install Sensor**

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/calendar.yaml
```

Make sure the sensor pod is created.

**4. Trigger Workflow**

Wait for 55 seconds to pass or if the sensor has `calendar-gateway:schedule` as event dependency, then wait for 30 min past each hour
for workflow to trigger.

## Add new schedule
Simply edit the gateway configmap, add the new endpoint entries and save. The gateway 
will automatically register the new cron schedules or intervals.

## Stop an existing schedule
Edit the gateway configmap, remove the entry and save it. Gateway pod will stop the schedule.
