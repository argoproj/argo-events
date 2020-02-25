# NATS Gateway & Sensor

NATS gateway listens to NATS subject notifications and helps sensor trigger the workloads.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/docs-gateway-setup/docs/assets/nats-setup.png?raw=true" alt="NATS Setup"/>
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
              "subject": "name_of_the_nats_subject",
              "body": "message_payload"
            }
        }


<br/>

## Setup

1. Make sure to have NATS cluster deployed in the Kubernetes. If you don't have one already installed, please refer https://github.com/nats-io/nats-operator for details.

   NATS cluster setup for test purposes,
   
        apiVersion: v1
        kind: Service
        metadata:
          name: nats
          namespace: argo-events
          labels:
            component: nats
        spec:
          selector:
            component: nats
          type: ClusterIP
          ports:
          - name: client
            port: 4222
          - name: cluster
            port: 6222
          - name: monitor
            port: 8222
        ---
        apiVersion: apps/v1beta1
        kind: StatefulSet
        metadata:
          name: nats
          namespace: argo-events
          labels:
            component: nats
        spec:
          serviceName: nats
          replicas: 1
          template:
            metadata:
              labels:
                component: nats
            spec:
              serviceAccountName: argo-events-sa
              containers:
              - name: nats
                image: nats:latest
                ports:
                - containerPort: 4222
                  name: client
                - containerPort: 6222
                  name: cluster
                - containerPort: 8222
                  name: monitor
                livenessProbe:
                  httpGet:
                    path: /
                    port: 8222
                  initialDelaySeconds: 10
                  timeoutSeconds: 5

2. Create the event source by running the following command. Make sure to update the appropriate fields.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/nats.yaml

3. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/nats.yaml

4. Inspect the gateway pod logs to make sure the gateway was able to subscribe to the subject specified in the event source to consume messages.

5. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/nats.yaml
        
6. Its time to publish a message for the subject specified in the event source. Refer the nats example to publish a message to the subject https://github.com/nats-io/go-nats-examples/tree/master/patterns/publish-subscribe.

7. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
