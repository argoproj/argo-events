# AWS Lambda Trigger

AWS Lambda provides a tremendous value but the event driven lambda invocation is limited to 
SNS, SQS and few other event sources. Argo Events makes it easy to integrate lambda with event sources
that are not native to AWS.

## Prerequisite
1. Set up the webhook gateway and event source.
2. Create a K8s secret that holds your access and secret key. Make sure to store the base64
   encoded keys in the secret.
3. Create a basic lambda function that can parse following payload,

   ```json
   {"name": "foo"}
   ```

## Lambda Trigger

The trigger specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#awslambdatrigger)

**Note**: You must declare the payload for the lambda trigger. Check out the HTTP trigger tutorial
to understand how to construct the payload.

1. Lets create a sensor with a lambda trigger

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/10-aws-lambda-trigger/sensor.yaml
   ```

2. Send a http request to webhook gateway,

   ```curl
   curl -d '{"name":"foo"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```

3. You should see the lambda execution in CloudWatch logs.

## Policy
To determine whether the function was successful or not, Lambda trigger provides a `Status` policy.
The `Status` holds a list of response statuses that are considered valid.

## Parameterization
Similar to HTTP trigger, the Lambda trigger provides `parameters` at both trigger resource and trigger template level.
