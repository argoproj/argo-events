# GCP PubSub

GCP PubSub gateway subscribes to messages published by GCP publisher and helps sensor trigger workloads.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/pubsub-setup.png?raw=true" alt="GCP PubSub Setup"/>
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
                    "id": "message id",
                    // Attributes represents the key-value pairs the current message
                    // is labelled with.
                    "attributes": "key-values",
                    "publishTime": "// The time at which the message was published",
                  	"body": "body refers to the message data",
                }
            }

## Setup

1. Fetch the project credentials JSON file from GCP console.

2. Create a K8s secret called `gcp-credentials` to store the credentials file

        apiVersion: v1
        data:
          key.json: <YOUR_CREDENTIALS_STRING_FROM_JSON_FILE>
        kind: Secret
        metadata:
          name: gcp-credentials
          namespace: argo-events
        type: Opaque

3. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/gateways/gcp-pubsub.yaml

4. Create the event source by running the following command.
   
           kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/gcp-pubsub.yaml

5. Inspect the gateway pod logs to make sure the gateway was able to subscribe to the topic specified in the event source to consume messages.

6. Create the sensor by running the following command,
   
           kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/gcp-pubsub.yaml

7. Publish a message from GCP PubSub console.

8. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
