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

Invoking the AWS Lambda without a request payload would not be very useful. The lambda trigger within a sensor
is invoked when sensor receives an event from a gateway. In order to construct a request payload based on the event data, sensor offers 
`payload` field as a part of the lambda trigger.

Let's examine a lambda trigger,

                awsLambda:
                  functionName: hello
                  accessKey:
                    name: aws-secret
                    key: accesskey
                  secretKey:
                    name: aws-secret
                    key: secretkey
                  namespace: argo-events
                  region: us-east-1
                  payload:
                    - src:
                        dependencyName: test-dep
                        dataKey: body.name
                      dest: name

The `payload` contains the list of `src` which refers to the source event and `dest` which refers to destination key within result request payload.

The `payload` declared above will generate a request payload like below,

        {
          "name": "foo" // name field from event data
        }

The above payload will be passed in the request to invoke the AWS lambda. You can add however many number of `src` and `dest` under `payload`. 

**Note**: Take a look at [Parameterization](https://argoproj.github.io/argo-events/tutorials/02-parameterization/) in order to understand how to extract particular key-value from
event data.

## Parameterization
Similar to other type of triggers, sensor offers parameterization for the AWS Lambda trigger. Parameterization is specially useful when
you want to define a generic trigger template in the sensor and populate values like funcation name, payload values on the fly.

Consider a scenario where you don't want to hard-code the function name and let the event data populate it.

                awsLambda:
                  functionName: hello // this will be replaced.
                  accessKey:
                    name: aws-secret
                    key: accesskey
                  secretKey:
                    name: aws-secret
                    key: secretkey
                  namespace: argo-events
                  region: us-east-1
                  payload:
                    - src:
                        dependencyName: test-dep
                        dataKey: body.message
                      dest: message
                  parameters:
                    - src:
                        dependencyName: test-dep
                        dataKey: body.function_name
                      dest: functionName

With `parameters` the sensor will replace the function name `hello` with the value of field `function_name` from event data.

You can learn more about trigger parameterization [here](https://argoproj.github.io/argo-events/tutorials/02-parameterization/).

## Policy
Trigger policy helps you determine the status of the lambda invocation and decide whether to stop or continue sensor. 

To determine whether the lamda was successful or not, Lambda trigger provides a `Status` policy.
The `Status` holds a list of response statuses that are considered valid.

                        awsLambda:
                          functionName: hello // this will be replaced.
                          accessKey:
                            name: aws-secret
                            key: accesskey
                          secretKey:
                            name: aws-secret
                            key: secretkey
                          namespace: argo-events
                          region: us-east-1
                          payload:
                            - src:
                                dependencyName: test-dep
                                dataKey: body.message
                              dest: message
                          policy:
                            status:
                                allow:
                                    - 200
                                    - 201

The above lambda trigger will be treated successful only if its invocation returns with either 200 or 201 status. 
