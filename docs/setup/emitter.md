# Emitter

Emitter gateway subscribes to a channel and helps sensor trigger the workloads.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/emitter-setup.png?raw=true" alt="Emitter Setup"/>
</p>

<br/>
<br/>

# Event Structure

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
              "topic": "name_of_the_topic",
              "body": "message_payload"
            }
        }

<br/>

## Setup

1. Deploy emitter in your local K8s cluster,

        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: broker
          labels:
            app: broker
        spec:
          clusterIP: None
          ports:
            - port: 4000
              targetPort: 4000
          selector:
            app: broker
        ---
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: broker
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: broker
          template:
            metadata:
              labels:
                app: broker
            spec:
              containers:
                - env:
                    - name: EMITTER_LICENSE
                      value: "zT83oDV0DWY5_JysbSTPTDr8KB0AAAAAAAAAAAAAAAI" # This is a test license, DO NOT USE IN PRODUCTION!
                    - name: EMITTER_CLUSTER_SEED
                      value: "broker"
                    - name: EMITTER_CLUSTER_ADVERTISE
                      value: "private:4000"
                  name: broker
                  image: emitter/server:latest
                  ports:
                    - containerPort: 8080
                    - containerPort: 443
                    - containerPort: 4000
                  volumeMounts:
                    - name: broker-volume
                      mountPath: /data
              volumes:
                - name: broker-volume
                  hostPath:
                    path: /emitter #directory on host
 
2. Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/emitter.yaml

3. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/emitter.yaml

4. Inspect the gateway pod logs to make sure the gateway was able to subscribe to the topic specified in the event source to consume messages.

5. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/emitter.yaml

6. Send message on emitter channel using one of the clients https://emitter.io/develop/golang/

7. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).


