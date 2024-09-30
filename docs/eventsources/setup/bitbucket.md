# Bitbucket (Cloud)

Bitbucket event-source programmatically configures webhooks for projects on Bitbucket and helps sensor trigger the workloads on events.

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
                   "body": "Body is the Bitbucket event payload",
                   "headers": "Headers from the Bitbucket event",
                }
            }

## Specification

Bitbucket event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.BitbucketEventSource). <br />
Example event-source yaml file is [here](https://github.com/argoproj/argo-events/blob/master/examples/event-sources/bitbucket.yaml).

## Setup

> **_NOTE:_** In this setup, we will use the basic auth strategy together with [App password](https://support.atlassian.com/bitbucket-cloud/docs/app-passwords/) (there is also support for [OAuth](https://support.atlassian.com/bitbucket-cloud/docs/use-oauth-on-bitbucket-cloud/)).

1.  Create an App password if you don't have one. Follow [instructions](https://support.atlassian.com/bitbucket-cloud/docs/app-passwords/) to create a new Bitbucket App password.
    Grant it the `Webhooks - Read and Write` permissions as well as any permissions that applies to the events that the webhook subscribes to (e.g. if you're using the [example event-source yaml file](https://github.com/argoproj/argo-events/blob/master/examples/event-sources/bitbucket.yaml) which subscribes to `repo:push` event then you would also need to grant the `Repositories - Read` permission).

1.  Base64 encode your App password and your Bitbucket username.

        echo -n <username> | base64
        echo -n <password> | base64

1.  Create a secret called `bitbucket-access` that contains your encoded Bitbucket credentials.

        apiVersion: v1
        kind: Secret
        metadata:
          name: bitbucket-access
        type: Opaque
        data:
          username: <base64-encoded-username-from-previous-step>
          password: <base64-encoded-password-from-previous-step>

1.  Deploy the secret into K8s cluster.

        kubectl -n argo-events apply -f bitbucket-access.yaml

1.  The event-source for Bitbucket creates a pod and exposes it via service.
    The name for the service is in `<event-source-name>-eventsource-svc` format.
    You will need to create an Ingress or OpenShift Route for the event-source service so that it can be reached from Bitbucket.
    You can find more information on Ingress or Route online.

1.  Create the event source by running the following command. You can use the [example event-source yaml file](https://github.com/argoproj/argo-events/blob/master/examples/event-sources/bitbucket.yaml) but make sure to replace the `url` field and to modify `owner`, `repositorySlug` and `projectKey` fields with your own repo.

        kubectl apply -n argo-events -f <event-source-file>

1.  Go to `Webhooks` under your project settings on Bitbucket and verify the webhook is registered. You can also do the same by looking at the event-source pod logs.

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/bitbucket.yaml

1.  Make a change to one of your project files and commit. It will trigger an argo workflow.

1.  Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
