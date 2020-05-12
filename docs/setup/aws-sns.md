# AWS SNS


SNS gateway subscribes to AWS SNS topics, listens events and helps sensor trigger workloads.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/aws-sns-setup.png?raw=true" alt="AWS SNS Setup"/>
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
                	"header": "sns headers",
                  	"body": "body refers to the sns notification data",
                }
            }


## Setup

1. Create a topic called `test` using aws cli or AWS SNS console.

2. Fetch your access and secret key for AWS account and base64 encode them.

3. Create a secret called `aws-secret` as follows,

        apiVersion: v1
        kind: Secret
        metadata:
          name: aws-secret
        type: Opaque
        data:
          accesskey: <base64-access-key>
          secretkey: <base64-secret-key>

4. Deploy the secret

        kubectl -n argo-events apply -f aws-secret.yaml

5. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/gateways/aws-sns.yaml

6. Wait for gateway pod to get into the running state.

7. Create an Ingress or Openshift Route for the gateway service to that it can be reached from AWS.
   You can find more information on Ingress or Route online.

8. Get the event source stored at https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/aws-sns.yaml

9. Change the `topicArn` and `url` under `webhook` to your gateway service url created in a previous step. Make sure this url is reachable from AWS.

8. Create the event source by running the following command.
   
        kubectl apply -n argo-events -f <event-source-file-updated-in-previous-step>

11. Go to SNS settings on AWS and verify the webhook is registered. You can also do the same by
    looking at the gateway pod logs.

12. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/aws-sns.yaml

13. Publish a message to the SNS topic and it will trigger an argo workflow.

14. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).

