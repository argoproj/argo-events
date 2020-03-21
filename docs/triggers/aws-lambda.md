# AWS Lambda

AWS Lambda provides a tremendous value but the event driven lambda invocation is limited to 
SNS, SQS and few other event sources. Argo Events makes it easy to integrate lambda with event sources
that are not native to AWS.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/cleanup-aws-lambda/docs/assets/aws-lambda-trigger.png?raw=true" alt="AWS Lambda Trigger"/>
</p>

<br/>
<br/>


## Trigger A Simple Lambda Function

1. Make sure your AWS account has permissions to execute Lambda. More info on AWS permissions is available
   [here](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_users_change-permissions.html).

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

4. Create a basic lambda function called `hello` either using AWS cli or console.

        exports.handler = async (event, context) => {
            console.log('name =', event.name);
            return event.name;
        };

5. Lets set up webhook gateway to invoke the lambda over http requests.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml
        
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml

6. Lets expose the webhook gateway using `port-forward` so that we can make a request to it.

        kubectl -n argo-events port-forward <name-of-gateway-pod> 12000:12000   

7. Deploy the webhook sensor with AWS Lambda trigger

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/aws-lambda-trigger.yaml

8. Once the sensor pod is in running state, make a `curl` request to webhook gateway,

         curl -d '{"name":"foo"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example 

9. It will trigger the AWS Lambda function `hello`. Look at the CloudWatch logs to verify.

## Specification

The AWS Lambda trigger specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#awslambdatrigger).

## Request Payload

The trigger specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#awslambdatrigger)

**Note**: You must declare the payload for the lambda trigger. Check out the HTTP trigger tutorial
to understand how to construct the payload.

1. Lets create a sensor with a lambda trigger

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/10-aws-lambda-trigger/sensor.yaml

2. Send a http request to webhook gateway,

        curl -d '{"name":"foo"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. You should see the lambda execution in CloudWatch logs.

## Policy
To determine whether the function was successful or not, Lambda trigger provides a `Status` policy.
The `Status` holds a list of response statuses that are considered valid.

## Parameterization
Similar to HTTP trigger, the Lambda trigger provides `parameters` at both trigger resource and trigger template level.

## OpenFaas Trigger
Similar to AWS lambda, you can trigger a OpenFaas function. The trigger specification is available [here](https://github.com/argoproj/argo-events/blob/worflow-triggers/api/sensor.md#openfaastrigger)
