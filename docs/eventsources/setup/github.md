# GitHub

GitHub event-source programmatically configures webhooks for projects on GitHub and helps sensor trigger the workloads on events.

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
                   "body": "Body is the Github event data",
                   "headers": "Headers from the Github event",
                }
            }

## Specification

GitHub event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.GithubEventSource). <br />
Example event-source yaml file is [here](https://github.com/argoproj/argo-events/blob/master/examples/event-sources/github.yaml).

## Setup

1.  Create an API token if you don't have one. Follow [instructions](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line) to create a new GitHub API Token.
    Grant it the `repo_hook` permissions.

1.  Base64 encode your API token key.

        echo -n <api-token-key> | base64

1.  Create a secret called `github-access` that contains your encoded GitHub API token. You can also include a secret key that is encoded with `base64` for your webhook if any.

        apiVersion: v1
        kind: Secret
        metadata:
          name: github-access
        type: Opaque
        data:
          token: <base64-encoded-api-token-from-previous-step>
          secret: <base64-encoded-webhook-secret-key>

1.  Deploy the secret into K8s cluster.

        kubectl -n argo-events apply -f github-access.yaml

1.  The event-source for GitHub creates a pod and exposes it via service.
    The name for the service is in `<event-source-name>-eventsource-svc` format.
    You will need to create an Ingress or OpenShift Route for the event-source service so that it can be reached from GitHub.
    You can find more information on Ingress or Route online.

1.  Create the event source by running the following command. Make sure to replace the `url` field.

        kubectl apply -n argo-events -f <event-source-file-updated-in-previous-step>

1.  Go to `Webhooks` under your project settings on GitHub and verify the webhook is registered. You can also do the same by looking at the event-source pod logs.

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/github.yaml

1.  Make a change to one of your project files and commit. It will trigger an argo workflow.

1.  Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
