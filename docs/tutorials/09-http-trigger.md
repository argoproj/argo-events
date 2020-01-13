# HTTP Trigger

In this tutorial, you will explore the HTTP triggers. 

Sometimes you face a situation where creating an Argo workflow on every event 
is not an ideal solution. This is where the HTTP trigger can help you. With this type of trigger, you can
connect any old/new API server with 20+ event sources supported by Argo Events or invoke serveless fucntions
without worrying about their respective event connector frameworks.

## Prerequisite
Set up the `Minio` gateway and event source. The K8s manifests are available under `examples/tutorials/09-http-trigger`.

## Connector for API servers
We will set up a basic go http server and connect it with the minio events.

1. Set up a simple http server that prints the request payload.

   ```bash
   kubectl -n argo-events apply -f 
   ```

## Payload Construction


## Connector for Serverless functions
