# Redis

Redis gateway subscribes to Redis publisher and helps sensor trigger workloads.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/redis-setup.png?raw=true" alt="Redis Setup"/>
</p>

<br/>
<br/>

## Event Structure

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
              	"channel": "Subscription channel",
              	"pattern": "Message pattern",
              	"body": "message body" // string
            }
        }

<br/>

## Setup

1. Follow the [documentation](https://kubernetes.io/docs/tutorials/configuration/configure-redis-using-configmap/#real-world-example-configuring-redis-using-a-configmap) to set up Redis database.

2. Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/redis.yaml

3. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/redis.yaml

4. Inspect the gateway pod logs to make sure the gateway was able to subscribe to the channel specified in the event source to consume messages.

5. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/redis.yaml

6. Log into redis pod using `kubectl`,

        kubectl -n argo-events exec -it <redis-pod-name> -c <redis-container-name> -- /bin/bash

7. Run `redis-cli` and publish a message on `FOO` channel.

        PUBLISH FOO hello

8. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
