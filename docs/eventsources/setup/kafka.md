# Kafka

Kafka event-source listens to messages on topics and helps the sensor trigger workloads.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

        {
            "context": {
              "type": "type_of_event_source",
              "specversion": "cloud_events_version",
              "source": "name_of_the_event_source",
              "id": "unique_event_id",
              "time": "event_time",
              "datacontenttype": "type_of_data",
              "subject": "name_of_the_configuration_within_event_source"
            },
            "data": {
              "topic": "kafka_topic",
              "partition": "partition_number",
              "body": "message_body",
              "timestamp": "timestamp_of_the_message"
            }
        }

## Specification

Kafka event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.KafkaEventSource).

## Setup

1.  Make sure to set up the Kafka cluster in Kubernetes if you don't already have one. You can refer to <https://github.com/Yolean/kubernetes-kafka> for installation instructions.

1.  Create the event source by running the following command. Make sure to update the appropriate fields.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/kafka.yaml

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/kafka.yaml

1.  Send message by using Kafka client. More info on how to send message at <https://kafka.apache.org/quickstart>.

1.  Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
