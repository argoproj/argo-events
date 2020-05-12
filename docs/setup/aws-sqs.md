# AWS SQS

SQS gateway listens to messages on AWS SQS queue and helps sensor trigger workloads.


<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/aws-sqs-setup.png?raw=true" alt="AWS SQS Setup"/>
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
                	"messageId": "message id",
                	// Each message attribute consists of a Name, Type, and Value. For more information,
                	// see Amazon SQS Message Attributes
                	// (https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-message-attributes.html)
                	// in the Amazon Simple Queue Service Developer Guide.
                	"messageAttributes": "message attributes", 
                  	"body": "Body is the message data",
                }
            }

<br/>

## Setup

1. Create a queue called `test` either using aws cli or AWS SQS management console.

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

2. Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/aws-sqs.yaml

3. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/gateways/aws-sqs.yaml

4. Inspect the gateway pod logs to make sure the gateway was able to subscribe to the queue specified in the event source to consume messages.

5. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/aws-sqs.yaml

6. Dispatch a message on sqs queue,

        aws sqs send-message --queue-url https://sqs.us-east-1.amazonaws.com/XXXXX/test --message-body '{"message": "hello"}'

7. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
