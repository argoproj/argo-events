# HTTP Trigger

Argo Events offers HTTP trigger which can easily invoke serverless functions like OpenFaaS, Kubeless, Knative, Nuclio and make REST API calls.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/http-trigger.png?raw=true" alt="HTTP Trigger"/>
</p>

<br/>
<br/>

## Specification

The HTTP trigger specification is available [here](../../APIs.md#argoproj.io/v1alpha1.HTTPTrigger).

## REST API Calls

Consider a scenario where your REST API server needs to consume events from event-sources S3, GitHub, SQS etc. Usually, you'd end up writing
the integration yourself in the server code, although server logic has nothing to do any of the event-sources. This is where Argo Events HTTP trigger
can help. The HTTP trigger takes the task of consuming events from event-sources away from API server and seamlessly integrates these events via REST API calls.

We will set up a basic go http server and connect it with the Minio events.

1.  The HTTP server simply prints the request body as follows.

        package main

        import (
         "fmt"
         "io"
         "net/http"
        )

        func hello(w http.ResponseWriter, req *http.Request) {
         body, err := io.ReadAll(req.Body)
         if err != nil {
          fmt.Printf("%+v\n", err)
          return
         }
         fmt.Println(string(body))
         fmt.Fprintf(w, "hello\n")
        }

        func main() {
         http.HandleFunc("/hello", hello)
         fmt.Println("server is listening on 8090")
         http.ListenAndServe(":8090", nil)
        }

2.  Deploy the HTTP server.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/09-http-trigger/http-server.yaml

3.  Create a service to expose the http server.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/09-http-trigger/http-server-svc.yaml

4.  Either use Ingress, OpenShift Route or port-forwarding to expose the http server.

        kubectl -n argo-events port-forward <http-server-pod-name> 8090:8090

5.  Our goals is to seamlessly integrate Minio S3 bucket notifications with REST API server created in previous step. So,
    lets set up the Minio event-source available [here](https://argoproj.github.io/argo-events/setup/minio/).
    Don't create the sensor as we will be deploying it in next step.

6.  Create a sensor as follows.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/http-trigger.yaml

7.  Now, drop a file onto `input` bucket in Minio server.

8.  The sensor has triggered a http request to the http server. Take a look at the logs.

        server is listening on 8090
        {"type":"minio","bucket":"input"}

9.  Great!!!

### Request Payload

In order to construct a request payload based on the event data, sensor offers
`payload` field as a part of the HTTP trigger.

Let's examine an HTTP trigger,

        http:
          url: http://http-server.argo-events.svc:8090/hello
          payload:
            - src:
                dependencyName: test-dep
                dataKey: notification.0.s3.bucket.name
              dest: bucket
            - src:
                dependencyName: test-dep
                contextKey: type
              dest: type
          method: POST  // GET, DELETE, POST, PUT, HEAD, etc.

The `payload` contains the list of `src` which refers to the source event and `dest` which refers to destination key within result request payload.

The `payload` declared above will generate a request payload like below,

        {
          "type": "type of event from event's context"
          "bucket": "bucket name from event data"
        }

The above payload will be passed in the HTTP request. You can add however many number of `src` and `dest` under `payload`.

**Note**: Take a look at [Parameterization](https://argoproj.github.io/argo-events/tutorials/02-parameterization/) in order to understand how to extract particular key-value from
event data.

### Parameterization

Similar to other type of triggers, sensor offers parameterization for the HTTP trigger. Parameterization is specially useful when
you want to define a generic trigger template in the sensor and populate values like URL, payload values on the fly.

You can learn more about trigger parameterization [here](https://argoproj.github.io/argo-events/tutorials/02-parameterization/).

### Policy

Trigger policy helps you determine the status of the HTTP request and decide whether to stop or continue sensor.

To determine whether the HTTP request was successful or not, the HTTP trigger provides a `Status` policy.
The `Status` holds a list of response statuses that are considered valid.

        http:
          url: http://http-server.argo-events.svc:8090/hello
          payload:
            - src:
                dependencyName: test-dep
                dataKey: notification.0s3.bucket.name
              dest: bucket
            - src:
                dependencyName: test-dep
                contextKey: type
              dest: type
          method: POST  // GET, DELETE, POST, PUT, HEAD, etc.
      retryStrategy:
        steps: 3
        duration: 3s
      policy:
        status:
          allow:
            - 200
            - 201

The above HTTP trigger will be treated successful only if the HTTP request returns with either 200 or 201 status.

## OpenFaaS

OpenFaaS offers a simple way to spin up serverless functions. Lets see how we can leverage Argo Events HTTP trigger
to invoke OpenFaaS function.

1.  If you don't have OpenFaaS installed, follow the [instructions](https://docs.openfaas.com/deployment/kubernetes/).

2.  Let's create a basic function. You can follow the [steps](https://blog.alexellis.io/serverless-golang-with-openfaas/).
    to set up the function.

         package function

         import (
          "fmt"
         )

         // Handle a serverless request
         func Handle(req []byte) string {
          return fmt.Sprintf("Hello, Go. You said: %s", string(req))
         }

3.  Make sure the function pod is up and running.

4.  We are going to invoke OpenFaaS function on a message on Redis Subscriber.

5.  Let's set up the Redis Database, Redis PubSub event-source as specified [here](https://argoproj.github.io/argo-events/setup/redis/).
    Do not create the Redis sensor, we are going to create it in next step.

6.  Let's create the sensor with OpenFaaS trigger.

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: redis-sensor
        spec:
          dependencies:
            - name: test-dep
              eventSourceName: redis
              eventName: example
          triggers:
            - template:
                name: openfaas-trigger
                http:
                  url: http://gateway.openfaas.svc.cluster.local:8080/function/gohash
                  payload:
                    - src:
                        dependencyName: test-dep
                      dest: bucket
                  method: POST

7.  Publish a message on `FOO` channel using `redis-cli`.

        PUBLISH FOO hello

8.  As soon as you publish the message, the sensor will invoke the OpenFaaS function `gohash`.

## Kubeless

Similar to REST API calls, you can easily invoke Kubeless functions using HTTP trigger.

1.  If you don't have Kubeless installed, follow the [installation](https://kubeless.io/docs/quick-start/).

2.  Lets create a basic function.

        def hello(event, context):
          print event
          return event['data']

3.  Make sure the function pod and service is created.

4.  Now, we are going to invoke the Kubeless function when a message is placed on a NATS queue.

5.  Let's set up the NATS event-source. Follow [instructions](https://argoproj.github.io/argo-events/setup/nats/#setup) for details.
    Do not create the NATS sensor, we are going to create it in next step.

6.  Let's create NATS sensor with HTTP trigger.

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: nats-sensor
        spec:
          dependencies:
            - name: test-dep
              eventSourceName: nats
              eventName: example
          triggers:
            - template:
                name: kubeless-trigger
                http:
                  serverURL: http://hello.kubeless.svc.cluster.local:8080
                  payload:
                    - src:
                        dependencyName: test-dep
                        dataKey: body.first_name
                      dest: first_name
                    - src:
                        dependencyName: test-dep
                        dataKey: body.last_name
                      dest: last_name
                  method: POST

7.  Once event-source and sensor pod are up and running, dispatch a message on `foo` subject using nats client.

        go run main.go -s localhost foo '{"first_name": "foo", "last_name": "bar"}'

8.  It will invoke Kubeless function `hello`.

        {'event-time': None, 'extensions': {'request': <LocalRequest: POST http://hello.kubeless.svc.cluster.local:8080/> }, 'event-type': None, 'event-namespace': None, 'data': '{"first_name":"foo","last_name":"bar"}', 'event-id': None}

# Other serverless frameworks

Similar to OpenFaaS and Kubeless invocation demonstrated above, you can easily trigger KNative, Nuclio, Fission functions using HTTP trigger.
