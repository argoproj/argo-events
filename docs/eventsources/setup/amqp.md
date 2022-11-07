# AMQP

AMQP event-source listens to messages on the MQ and helps sensor trigger the workloads.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

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
               "contentType": "ContentType is the MIME content type",
               "contentEncoding": "ContentEncoding is the MIME content encoding",
               "deliveryMode": "Delivery mode can be either - non-persistent (1) or persistent (2)",
               "priority": "Priority refers to the use - 0 to 9",
               "correlationId": "CorrelationId is the correlation identifier",
               "replyTo": "ReplyTo is the address to reply to (ex: RPC)",
               "expiration": "Expiration refers to message expiration spec",
               "messageId": "MessageId is message identifier",
               "timestamp": "Timestamp refers to the message timestamp",
               "type": "Type refers to the message type name",
               "appId": "AppId refers to the application id",
               "exchange": "Exchange is basic.publish exchange",
               "routingKey": "RoutingKey is basic.publish routing key",
               "body": "Body represents the message body",
            }
        }

<br/>

## Setup

1. Lets set up RabbitMQ locally.

        apiVersion: v1
        kind: Service
        metadata:
          labels:
            component: rabbitmq
          name: rabbitmq-service
        spec:
          ports:
            - port: 5672
          selector:
            app: taskQueue
            component: rabbitmq
        ---
        apiVersion: v1
        kind: ReplicationController
        metadata:
          labels:
            component: rabbitmq
          name: rabbitmq-controller
        spec:
          replicas: 1
          template:
            metadata:
              labels:
                app: taskQueue
                component: rabbitmq
            spec:
              containers:
                - image: rabbitmq
                  name: rabbitmq
                  ports:
                    - containerPort: 5672
                  resources:
                    limits:
                      cpu: 100m

2. Make sure the RabbitMQ controller pod is up and running before proceeding further.

3. Expose the RabbitMQ server to local publisher using `port-forward`.

        kubectl -n argo-events port-forward <rabbitmq-pod-name> 5672:5672

4. Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/amqp.yaml

6. Inspect the event-source pod logs to make sure it was able to subscribe to the exchange specified in the event source to consume messages.

7. Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/amqp.yaml

8. Lets set up a RabbitMQ publisher. If you don't have `pika` installed, run.

        python -m pip install pika --upgrade

9. Open a python REPL and run following code to publish a message on `exchange` called `test`.

        import pika
        connection = pika.BlockingConnection(pika.ConnectionParameters('localhost'))
        channel = connection.channel()
        channel.basic_publish(exchange='test',
                              routing_key='hello',
                              body='{"message": "hello"}')

10. As soon as you publish a message, sensor will trigger an Argo workflow. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
