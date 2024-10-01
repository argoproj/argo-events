# GCP Pub/Sub

GCP Pub/Sub event-source subscribes to messages published by GCP publisher and helps sensor trigger workloads.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

```js
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
        "id": "message id",
        // Attributes represents the key-value pairs the current message
        // is labelled with.
        "attributes": "key-values",
        "publishTime": "// The time at which the message was published",
        "body": "body refers to the message data",
    }
}
```

## Specification

GCP Pub/Sub event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.PubSubEventSource).

## Setup

1.  Fetch the project credentials JSON file from GCP console.

    If you use Workload Identity, you can skip this and next steps.

1.  Create a K8s secret called `gcp-credentials` to store the credentials file.

        apiVersion: v1
        data:
          key.json: <YOUR_CREDENTIALS_STRING_FROM_JSON_FILE>
        kind: Secret
        metadata:
          name: gcp-credentials
          namespace: argo-events
        type: Opaque

1.  Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/gcp-pubsub.yaml

    If you use Workload Identity, omit `credentialSecret` field. Instead don't forget to configure appropriate service account (see [example](https://github.com/argoproj/argo-events/blob/stable/examples/event-sources/gcp-pubsub.yaml)).

1.  Inspect the event-source pod logs to make sure it was able to subscribe to the topic.

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/gcp-pubsub.yaml

1.  Publish a message from GCP Pub/Sub console.

1.  Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Subscription, topic and service account preparation

You can use existing subscriptions/topics, or let Argo Events create them.

Here's the table of which fields are required in the configuration file and what permissions are needed for service account.

| Actions                                    | Required configuration fields                           | Necessary permissions for service account                                                                    | Example role                                         |
| :----------------------------------------- | :------------------------------------------------------ | :----------------------------------------------------------------------------------------------------------- | :--------------------------------------------------- |
| Use existing subscription                  | Existing `SubscriptionID`                               | `pubsub.subscriptions.consume` for the subscription                                                          | `roles/pubsub.subscriber`                            |
| Use existing subscription and verify topic | Existing `SubscriptionID` and its `Topic`               | Above +<br>`pubsub.subscriptions.get` for the subscription                                                   | `roles/pubsub.subscriber`<br>+ `roles/pubsub.viewer` |
| Create subscription for existing topic     | Existing `Topic`<br>(`SubscriptionID` is optional†)     | Above +<br>`pubsub.subscriptions.create` for the project<br>`pubsub.topics.attachSubscription` for the topic | `roles/pubsub.subscriber`<br>+ `roles/pubsub.editor` |
| Create topic and subscription              | Non-existing `Topic`<br>(`SubscriptionID` is optional†) | Above +<br>`pubsub.topic.create` for the project                                                             | `roles/pubsub.subscriber`<br>+ `roles/pubsub.editor` |

† _If you omit `SubscriptionID`, a generated hash value is used._

For more details about access control, refer to GCP documents:

- [Access control  |  Cloud Pub/Sub Documentation  |  Google Cloud ⧉](https://cloud.google.com/pubsub/docs/access-control#testing_permissions)

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
