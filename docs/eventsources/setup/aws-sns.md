# AWS SNS

SNS event-source subscribes to AWS SNS topics, listens events and helps sensor trigger the workloads.

## Event Structure

The structure of an event dispatched by the event-source over eventbus looks like following,

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
                 "header": "sns headers",
                   "body": "body refers to the sns notification data",
                }
            }

## Setup

1. Create a topic called `test` using aws cli or AWS SNS console.

1. Fetch your access and secret key for AWS account and base64 encode them.

1. Create a secret called `aws-secret` as follows.

        apiVersion: v1
        kind: Secret
        metadata:
          name: aws-secret
        type: Opaque
        data:
          accesskey: <base64-access-key>
          secretkey: <base64-secret-key>

1. Deploy the secret.

        kubectl -n argo-events apply -f aws-secret.yaml

1. The event-source for AWS SNS creates a pod and exposes it via service.
   The name for the service is in `<event-source-name>-eventsource-svc` format.
   You will need to create an Ingress or OpenShift Route for the event-source service so that it can be reached from AWS.
   You can find more information on Ingress or Route online.

1. Create the event source by running the following command. Make sure to update the URL in the configuration within the event-source.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/aws-sns.yaml

1. Go to SNS settings on AWS and verify the webhook is registered. You can also check it by inspecting the event-source pod logs.

1. Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/aws-sns.yaml

1. Publish a message to the SNS topic, and it will trigger an argo workflow.

1. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
