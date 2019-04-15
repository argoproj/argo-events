# Calendar

Calendar gateway can be used when you to schedule K8s resources on an interval
or cron schedule.

## How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/core/calendar/config.go#L35-L55).

The [gateway configmap](../../../examples/gateways/calendar-gateway-configmap.yaml) contains three event source configurations,

The `interval` configuration will basically generate an event after every 10 seconds.

The `schedule` configuration will generate an event every 30 minutes past an hour

The `withdata` configuration will generate an event every 30 minutes past an hour but the time zone is "America/New_York"
and along with event time, event payload will also contain a string  "{\r\n\"hello\": \"world\"\r\n}"

**What is the use of `userPayload`?**

Only the event time is probably not sufficient as event payload. You may want to have an accompanying user defined data.
This user defined data is `userPayload`. You can basically place any string here and gateway will put that in event payload.  


### Event Payload Structure
The [event payload](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/core/calendar/config.go#L60-L64) contains `eventTime` and `userPayload` (optional)

## Setup
**1. Install [Gateway Configmap](../../../examples/gateways/calendar-gateway-configmap.yaml)**

**2. Install [Gateway](../../../examples/gateways/calendar.yaml)**

Make sure the gateway pod is created.

**3. Install [Sensor](../../../examples/sensors/calendar.yaml)**

Make sure the sensor pod is created.

**4. Trigger Workflow**

Wait for 10 seconds to pass or if the sensor has `calendar-gateway:schedule` as event dependency, then wait for 30 min past each hour
for workflow to trigger.

## Add new schedule
Simply edit the gateway configmap, add the new endpoint entries and save. The gateway 
will automatically register the new cron schedules or intervals.

## Stop an existing schedule
Edit the gateway configmap, remove the entry and save it. Gateway pod will stop the schedule.
