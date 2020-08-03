# NATS Trigger

NATS trigger allows sensor to publish events on NATS subjects. This trigger helps source the events from outside world into your messaging queues.

## Specification
The NATS trigger specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#natstrigger).

## Walkthrough

1. Consider a scenario where you are expecting a file drop onto a Minio bucket and want to place that event
   on a NATS subject.

1. Set up the Minio Event Source [here](https://argoproj.github.io/argo-events/setup/minio/). 
   Do not create the Minio sensor, we are going to create it in next step.
   
1. Lets create the sensor,

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: minio-sensor
        spec:
          dependencies:
            - name: test-dep
              eventSourceName: minio
              eventName: example
          subscription:
            http:
              port: 9300
          triggers:
            - template:
                name: nats-trigger
                nats:
                  # NATS Server URL
                  url: nats.argo-events.svc:4222
                  # Name of the subject
                  subject: minio-events
                  payload:
                    - src:
                        dependencyName: test-dep
                        dataKey: notification.0.s3.object.key
                      dest: fileName
                    - src:
                        dependencyName: test-dep
                        dataKey: notification.0.s3.bucket.name
                      dest: bucket

1. The NATS message needs a body. In order to construct message based on the event data, sensor offers 
   `payload` field as a part of the NATS trigger.

   The `payload` contains the list of `src` which refers to the source event and `dest` which refers to destination key within result request payload.

   The `payload` declared above will generate a message body like below,

        {
            "fileName": "hello.txt" // name/key of the object
            "bucket": "input" // name of the bucket
        }

1. If you are running NATS on local K8s cluster, make sure to `port-forward` to pod,

        kubectl -n argo-events port-forward <nats-pod-name> 4222:4222
        
1. Subscribe to the subject called `minio-events`. Refer the nats example to publish a message to the subject https://github.com/nats-io/go-nats-examples/tree/master/patterns/publish-subscribe.
   
           go run main.go -s localhost minio-events'

1. Drop a file called `hello.txt` onto the bucket `input` and you will receive the message on NATS subscriber
   as follows,
   
        [#1] Received on [minio-events]: '{"bucket":"input","fileName":"hello.txt"}'
