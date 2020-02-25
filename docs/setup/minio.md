# Minio Gateway & Sensor

Minio gateway listens to minio bucket notifications and helps sensor trigger the workloads.

**_Note: Minio gateway is exclusive for the Minio server. If you want to trigger workloads on AWS S3 bucket notification,
set up the AWS SNS gateway._**

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/docs-gateway-setup/docs/assets/minio-setup.png?raw=true" alt="Minio Setup"/>
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
              notification: [
                {
                  /* Minio notification. More info is available at https://docs.min.io/docs/minio-bucket-notification-guide.html
                }
              ]
            }
        }

<br/>

## Setup

1. Make sure to have minio server deployed and reachable from the gateway. More info on minio server setup 
is available at https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/k8s-yaml.md.

2. Create a K8s secret that holds the access and secret key. This secret will be referred in the minio event source definition that we are going to install in a later step.

        apiVersion: v1
        data:
          # base64 of minio
          accesskey: bWluaW8=
          # base64 of minio123
          secretkey: bWluaW8xMjM=
        kind: Secret
        metadata:
          name: artifacts-minio
          namespace: argo-events

2. Install gateway in the `argo-events` namespace using following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/minio.yaml

   Once the gateway resource is created, the gateway controller will process it and create a pod.
   
   If you don't see the pod in `argo-events` namespace, check the gateway controller logs
   for errors.

3. **Make sure to create the bucket called `input`. If you don't have the bucket created, then the gateway will mark the event source as failure.**

4. If the minio service name differs from the event source, then make sure to update the event source.

5. If you inspect the gateway resource definition, you will notice that it refers to the event source `minio-event-source`. Lets install event source in the `argo-events` namespace,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/minio.yaml
   
6. Check the gateway logs to make sure the gateway has processed the event source.

7. Lets create the sensor,
   
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/minio.yaml   

8. Create a file named and `hello-world.txt` and upload it onto to the bucket. This will trigger the argo workflow.

9. Run `argo list` to find the workflow.

<br/>

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
