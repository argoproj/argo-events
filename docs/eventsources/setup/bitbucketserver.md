# Bitbucket Server

Bitbucket Server event-source programmatically configures webhooks for projects on Bitbucket Server and helps sensor trigger the workloads on events.

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
                   "body": "Body is the Bitbucket Server event payload",
                   "headers": "Headers from the Bitbucket Server event",
                }
            }

## Specification

Bitbucket Server event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.BitbucketServerEventSource). <br />
Example event-source yaml file is [here](https://github.com/argoproj/argo-events/blob/master/examples/event-sources/bitbucketserver.yaml).

## Setup

1.  Create an API token if you don't have one. Follow [instructions](https://confluence.atlassian.com/bitbucketserver072/personal-access-tokens-1005335924.html) to create a new Bitbucket Server API Token.
    Grant it the `Projects: Admin` permissions.

1.  Base64 encode your API token key.

        echo -n <api-token-key> | base64

1.  Create a secret called `bitbucketserver-access` that contains your encoded Bitbucket Server API token. You can also include a secret key that is encoded with `base64` for your webhook if any.

        apiVersion: v1
        kind: Secret
        metadata:
          name: bitbucketserver-access
        type: Opaque
        data:
          token: <base64-encoded-api-token-from-previous-step>
          secret: <base64-encoded-webhook-secret-key>

1.  Deploy the secret into K8s cluster.

        kubectl -n argo-events apply -f bitbucketserver-access.yaml

1.  The event-source for Bitbucket Server creates a pod and exposes it via service.
    The name for the service is in `<event-source-name>-eventsource-svc` format.
    You will need to create an Ingress or Openshift Route for the event-source service so that it can be reached from Bitbucket Server.
    You can find more information on Ingress or Route online.

1.  Create the event source by running the following command. You can use the example event-source yaml file from [here](https://github.com/argoproj/argo-events/blob/master/examples/event-sources/bitbucketserver.yaml) but make sure to replace the `url` field and to modify the `repositories` list with your own repos.

        kubectl apply -n argo-events -f <event-source-file>

1.  Go to `Webhooks` under your project settings on Bitbucket Server and verify the webhook is registered. You can also do the same by looking at the event-source pod logs.

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/bitbucketserver.yaml

1.  Make a change to one of your project files and commit. It will trigger an argo workflow.

1.  Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
