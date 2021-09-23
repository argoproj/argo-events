# Pulsar Trigger

Pulsar trigger allows sensor to publish events on Pulsar topic. This trigger helps source the events from outside world into your messaging queues.

## Specification

The Pulsar trigger specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#pulsartrigger).

## Walkthrough

1. Consider a scenario where you are expecting a file drop onto a Minio bucket and want to place that event on a Pulsar topic.

1. Set up the Minio Event Source [here](https://argoproj.github.io/argo-events/setup/minio/). 
   Do not create the Minio sensor, we are going to create it in next step.
   
1. Lets create the sensor.

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: minio-sensor
        spec:
          dependencies:
            - name: test-dep
              eventSourceName: minio
              eventName: example
          triggers:
            - template:
                name: pulsar-trigger
                pulsar:
                  # Pulsar URL
                  url: pulsar://pulsar.argo-events.svc:6650
                  # Name of the topic
                  topic: minio-events
                  payload:
                    - src:
                        dependencyName: test-dep
                        dataKey: notification.0.s3.object.key
                      dest: fileName
                    - src:
                        dependencyName: test-dep
                        dataKey: notification.0.s3.bucket.name
                      dest: bucket

1. The Pulsar message needs a body. In order to construct message based on the event data, sensor offers 
   `payload` field as a part of the Pulsar trigger.

   The `payload` contains the list of `src` which refers to the source event and `dest` which refers to destination key within result request payload.

   The `payload` declared above will generate a message body like below.

        {
            "fileName": "hello.txt" // name/key of the object
            "bucket": "input" // name of the bucket
        }

1. Drop a file called `hello.txt` onto the bucket `input` and you will receive the message on Pulsar topic
