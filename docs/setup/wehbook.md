# Webhook Gateway & Sensor

Webhook gateway exposes a http server and allows external entities to trigger workloads via
http requests.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/setup/docs/assets/webhook-gateway.png?raw=true" alt="Webhook Gateway"/>
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
              "header": {/* the headers from the request received by the gateway from the external entity */},
              "body": { /* the payload of the request received by the gateway from the external entity */},
            }
        }

<br/>

## Setup

1. Install gateway in the `argo-events` namespace using following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml

   Once the gateway resource is created, the gateway controller will process it and create a pod and a service.
   
   If you don't see the pod and service in `argo-events` namespace, check the gateway controller logs
   for errors.

2. If you inspect the gateway resource definition, you will notice it points to the event source called
   `webhook-event-source`. Lets install event source in the `argo-events` namespace,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml
   
3. Check the gateway logs to make sure the gateway has processed the event source.

4. The gateway is now listening for HTTP requests on port `12000` and endpoint `/example`.
    Its time to create the sensor,
    
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/webhook.yaml   

5. Once the sensor pod is in running state, test the setup by sending a POST request to gateway service.

<br/>

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
