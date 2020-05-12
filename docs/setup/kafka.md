# Kafka

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/kafka-setup.png?raw=true" alt="KAFKA Setup"/>
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
              "topic": "kafka_topic",
              "partition": "partition_number",
              "body": "message_body",
              "timestamp": "timestamp_of_the_message"
            }
        }

<br/>

## Setup

1. Make sure to setup the Kafka cluster in Kubernetes if you don't already have one. You can refer to https://github.com/Yolean/kubernetes-kafka
for installation instructions.

2. Create the event source by running the following command. Make sure to update the appropriate fields.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/kafka.yaml

3. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/kafka.yaml

4. Inspect the gateway pod logs to make sure the gateway was able to subscribe to the topic specified in the event source to consume messages.

5. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/kafka.yaml

6. Send message by using Kafka client. More info on how to send message at https://kafka.apache.org/quickstart

7. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
