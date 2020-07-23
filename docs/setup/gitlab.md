# GitLab

GitLab event-source programmatically configures webhooks for projects on GitLab and helps sensor trigger the workloads upon events.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

            {
                "context": {
                  "type": "type_of_event_source",
                  "specVersion": "cloud_events_version",
                  "source": "name_of_the_event_source",
                  "eventID": "unique_event_id",
                  "time": "event_time",
                  "dataContentType": "type_of_data",
                  "subject": "name_of_the_configuration_within_event_source"
                },
                "data": {
                  	"body": "Body is the gitlab event data",
                  	"headers": "Headers from the Gitlab event",
                }
            }

## Specification

GitLab event-source specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/event-source.md#gitlabeventsource).

## Setup

1. Create an API token if you don't have one. Follow [instructions](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) to create a new GitLab API Token.
   Grant it the `api` permissions.

1. Base64 encode your api token key,

        echo -n <api-token-key> | base64

1. Create a secret called `gitlab-access`.

        apiVersion: v1
        kind: Secret
        metadata:
          name: gitlab-access
        type: Opaque
        data:
          token: <base64-encoded-api-token-from-previous-step>

1. Deploy the secret into K8s cluster.

        kubectl -n argo-events apply -f gitlab-access.yaml

1. The event-source for GitLab creates a pod and exposes it via service.
   The name for the service is in `<event-source-name>-eventsource-svc` format.
   You will need to create an Ingress or Openshift Route for the event-source service so that it can be reached from GitLab.
   You can find more information on Ingress or Route online.

1. Create the event source by running the following command. Make sure to update `url` field.
   
        kubectl apply -n argo-events -f <event-source-file-updated-in-previous-step>

1. Go to `Webhooks` under your project settings on GitLab and verify the webhook is registered.
    
1. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/gitlab.yaml

1. Make a change to one of your project files and commit. It will trigger an argo workflow.

1. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).

