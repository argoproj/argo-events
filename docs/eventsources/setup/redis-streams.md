# Redis Streams

Redis stream event-source listens to messages on Redis streams and helps sensor trigger workloads.

Messages from the stream are read using the Redis consumer group. The main reason for using consumer group is to resume from the last read upon pod restarts. A common consumer group (defaults to "argo-events-cg") is created (if not already exists) on all specified streams. When using consumer group, each read through a consumer group is a write operation, because Redis needs to update the last retrieved message id and the pending entries list(PEL) of that specific user in the consumer group. So it can only work with the master Redis instance and not replicas (<https://redis.io/topics/streams-intro>).

Redis stream event source expects all the streams to be present on the Redis server. This event source only starts pulling messages from the streams when all of the specified streams exist on the Redis server. On the initial setup, the consumer group is created on all the specified streams to start reading from the latest message (not necessarily the beginning of the stream). On subsequent setups (the consumer group already exists on the streams) or during pod restarts, messages are pulled from the last unacknowledged message in the stream.

The consumer group is never deleted automatically. If you want a completely fresh setup again, you must delete the consumer group from the streams.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

        {
            "context": {
              "id": "unique_event_id",
              "source": "name_of_the_event_source",
              "specversion": "cloud_events_version",
              "type": "type_of_event_source",
              "datacontenttype": "type_of_data",
              "subject": "name_of_the_configuration_within_event_source",
              "time": "event_time"
            },
            "data": {
               "stream": "Name of the Redis stream",
               "message_id": "Message Id",
               "values": "message body"
            }
        }

Example:

        {
          "context": {
            "id": "64313638396337352d623565612d343639302d383262362d306630333562333437363637",
            "source": "redis-stream",
            "specversion": "1.0",
            "type": "redisStream",
            "datacontenttype": "application/json",
            "subject": "example",
            "time": "2022-03-17T04:47:42Z"
          },
          "data": {
                  "stream":"FOO",
                  "message_id":"1647495121754-0",
                  "values": {"key-1":"val-1", "key-2":"val-2"}
          }
        }

## Specification

Redis stream event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.RedisStreamEventSource).

## Setup

1.  Follow the [documentation](https://kubernetes.io/docs/tutorials/configuration/configure-redis-using-configmap/#real-world-example-configuring-redis-using-a-configmap) to set up Redis database.

1.  Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/redis-streams.yaml

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/redis-streams.yaml

1.  Log into redis pod using `kubectl`.

        kubectl -n argo-events exec -it <redis-pod-name> -c <redis-container-name> -- /bin/bash

1.  Run `redis-cli` and publish a message on the stream `FOO`.

        XADD FOO * message hello

1.  Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Troubleshoot

Redis stream event source expects all the streams to be present on redis server. It only starts pulling messages from the streams when all of the specified streams exist on the redis server.

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
