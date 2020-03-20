# MQTT

MQTT gateway listens to messages on from IoT devices and helps sensor trigger the workloads.  

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/mqtt-setup.png?raw=true" alt="MQTT Setup"/>
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
                "topic": "Topic refers to the MQTT topic name",
                "messageId": "MessageId is the unique ID for the message",
                "body": "Body is the message payload"
            }
        }

<br/>

## Setup

1. Make sure to setup the MQTT Broker and Bridge in Kubernetes if you don't already have one. 

2. Create the event source by running the following command. Make sure to update the appropriate fields.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/mqtt.yaml

3. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/mqtt.yaml

4. Inspect the gateway pod logs to make sure the gateway was able to subscribe to the topic specified in the event source to consume messages.

5. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/mqtt.yaml

6. Send message by using MQTT client.

7. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).


