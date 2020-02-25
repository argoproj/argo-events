# Calendar Gateway & Sensor

Calendar gateway generates events on either a cron schedule or an interval and help trigger workloads.

 
<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/calendar-setup.png?raw=true" alt="Calendar Setup"/>
</p>

<br/>
<br/>

## Event Structure
The structure of an event dispatched by the gateway to the sensor looks like following,

        {
            "context": {
              "type": "type_of_gateway",
              "specVersion": "cloud_events_version",
              "source": "name_of_the_gateway",
              "eventID": "unique_event_id",
              "time": "event_time",
              "dataContentType": "type_of_data",
              "subject": "name_of_the_event_within_event_source"
            },
            "data": {
              "eventTime": {/* UTC time of the event */},
              "userPayload": { /* static payload available in the event source */},
            }
        }

<br/>

## Setup

1. Install gateway in the `argo-events` namespace using following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/calendar.yaml

   Once the gateway resource is created, the gateway controller will process it and create a pod.
   
   If you don't see the pod in `argo-events` namespace, check the gateway controller logs
   for errors.

2. If you inspect the gateway resource definition, you will notice it points to the event source called
   `calendar-event-source`. Lets install event source in the `argo-events` namespace,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/calendar.yaml
   
3. Check the gateway logs to make sure the gateway has processed the event source.

4. The gateway will generate events at every 10 seconds. Lets create the sensor,
    
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/calendar.yaml   

5. Once the sensor pod is in running state, wait for next interval to occur.

<br/>

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
