# Redis

Redis event-source subscribes to Redis publisher and helps sensor trigger workloads.

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
               "channel": "Subscription channel",
               "pattern": "Message pattern",
               "body": "message body" // string
            }
        }

## Specification

Redis event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.RedisEventSource).

## Setup

1.  Follow the [documentation](https://kubernetes.io/docs/tutorials/configuration/configure-redis-using-configmap/#real-world-example-configuring-redis-using-a-configmap) to set up Redis database.

1.  Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/redis.yaml

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/redis.yaml

1.  Log into redis pod using `kubectl`.

        kubectl -n argo-events exec -it <redis-pod-name> -c <redis-container-name> -- /bin/bash

1.  Run `redis-cli` and publish a message on `FOO` channel.

        PUBLISH FOO hello

1.  Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
