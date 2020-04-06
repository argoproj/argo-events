# Event-Driven Parameterized Jupyter Notebooks

Jupyter notebooks are prevalent in data science community to develop models, run analysis and generate reports, etc.
But in many situations, a data scientist must feed varying parameters to tune the notebook to generate an optimal model. 
Tools like [papermill](https://papermill.readthedocs.io/en/latest/) makes it easy to parameterize the notebook and Argo Events makes it super easy
to set up **event-driven** parameterized notebooks.

# Prerequisites
1. Install [Argo Workflows](https://github.com/argoproj/argo/blob/master/docs/getting-started.md).
1. Install [Argo Events](https://argoproj.github.io/argo-events/installation/).
1. Install NATS,
    
        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/nats-deploy.yaml

1. Port forward to NATS pod,

        kubectl -n argo-events port-forward <nats-pod-name> 4222:4222

1. Install Minio,

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/artifact-minio.yaml

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/minio-deploy.yaml

1. Port forward to Minio pod,

        kubectl -n argo-events port-forward <minio-pod-name> 9000:9000


# Setup
In this demo, we are going to set up an image processing pipeline using 2 notebooks. Lets consider the ArgoProj icon image

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/argo.png?raw=true" alt="Argo"/>
</p>

<br/>
<br/>

1. The [first notebook](https://github.com/VaibhavPage/argo-events-demo/blob/master/img-out.ipynb) will take the clean ArgoProj logo and add noise to it.
2. The [second notebook](https://github.com/VaibhavPage/argo-events-demo/blob/master/matcher.ipynb) is going to determine the similarity between clean image and the image with noise.
   If the match is > 80%, then model is optimal, else we need to tune the noise parameters.

<br/>

<p align="center">
  <img src="https://github.com/VaibhavPage/argo-events-demo/blob/master/out/argo-demo.png?raw=true" alt="Argo Demo"/>
</p>

<br/>

3. We will set up two gateways, Webhook and Minio. The webhook gateway will listen to HTTP requests to tune the 
   notebook to add noise to image. The notebook will store the noisy image to Minio.
   
4. The minio gateway will listen to file drop events for a specific bucket. Once the noisy image is dropped into that bucket,
   we will run the second notebook that determines the similarity of images.
   
<br/>

<p align="center">
  <img src="https://github.com/VaibhavPage/argo-events-demo/blob/master/argo-events-demo-pipeline.png?raw=true" alt="Argo Demo"/>
</p>

<br/>

5. Create webhook event source. It consist configuration for gateway to listen for HTTP POST requests on port 12000.


        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/webhook-event-source.yaml

6. Create webhook gateway,

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/webhook-gateway.yaml
        
1. Port forward to webhook gateway pod,

        kubectl -n argo-events port-forward <webhook-gateway-pod-name> 12000:12000
        
7. Create webhook sensor,

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/webhook-sensor.yaml

8. Lets inspect webhook sensor,

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: webhook-sensor
          labels:
            sensors.argoproj.io/sensor-controller-instanceid: argo-events
        spec:
          template:
            spec:
              containers:
                - name: sensor
                  image: argoproj/sensor:v0.13.0-rc
                  imagePullPolicy: Always
              serviceAccountName: argo-events-sa
          dependencies:
            - name: test-dep
              gatewayName: webhook-gateway
              eventName: example
          subscription:
            http:
              port: 9300
          triggers:
            - template:
                name: webhook-workflow-trigger
                k8s:
                  group: argoproj.io
                  version: v1alpha1
                  resource: workflows
                  operation: create
                  source:
                    resource:
                      apiVersion: argoproj.io/v1alpha1
                      kind: Workflow
                      metadata:
                        generateName: noisy-processor-
                      spec:
                        entrypoint: noisy
                        arguments:
                          parameters:
                            - name: filterA
                              value: "5"
                            - name: filterB
                              value: "5"
                            - name: sVSp
                              value: "0.5"
                            - name: amount
                              value: "0.004"
                        templates:
                          - name: noisy
                            serviceAccountName: argo-events-sa
                            inputs:
                              parameters:
                                - name: filterA
                                - name: filterB
                                - name: sVSp
                                - name: amount
                            container:
                              image: metalgearsolid/demo-blur-argo-logo:latest
                              command: [papermill]
                              imagePullPolicy: Always
                              env:
                                - name: AWS_ACCESS_KEY_ID
                                  value: minio
                                - name: AWS_SECRET_ACCESS_KEY
                                  value: minio123
                                - name: AWS_DEFAULT_REGION
                                  value: us-east-1
                                - name: BOTO3_ENDPOINT_URL
                                  value: http://minio-service.argo-events.svc:9000
                              args:
                                - "noise.ipynb"
                                - "s3://output/noisy-out.ipynb"
                                - "-p"
                                - "filterA"
                                - "{{inputs.parameters.filterA}}"
                                - "-p"
                                - "filterB"
                                - "{{inputs.parameters.filterB}}"
                                - "-p"
                                - "sVSp"
                                - "{{inputs.parameters.sVSp}}"
                                - "-p"
                                - "amount"
                                - "{{inputs.parameters.amount}}"
                  parameters:
                    - src:
                        dependencyName: test-dep
                        dataKey: body.filterA
                      dest: spec.arguments.parameters.0.value
                    - src:
                        dependencyName: test-dep
                        dataKey: body.filterB
                      dest: spec.arguments.parameters.1.value
                    - src:
                        dependencyName: test-dep
                        dataKey: body.sVSp
                      dest: spec.arguments.parameters.2.value
                    - src:
                        dependencyName: test-dep
                        dataKey: body.amount
                      dest: spec.arguments.parameters.3.value

1. The sensor trigger is an Argo workflow that runs a jupyter notebook with papermill.
   It takes arguments for Guassian filter and Slat+Pepper noise in addition to S3 configuration.
   The event data received from HTTP POST request is made to override the arguments to workflow on the fly.

1. Lets configurre Minio client mc,

         mc config host add minio http://localhost:9000 minio minio123

1. Create a bucket on Minio called `output`.

        mc mb minio/output

1. Create the Minio event source that makes the gateway listen to file events for `output` bucket,

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/minio-event-source.yaml
        
1. Create Minio gateway

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/minio-gateway.yaml

1. Create Minio sensor,

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/VaibhavPage/argo-events-demo/master/minio-sensor.yaml
        
1. The Minio sensor triggers an Argo workflow that determines the similarity index of the image that was put onto `noisy-bucket` on Minio
   and the original Argo logo. If the match is less than 80%, it publishes failure message on NATS subject called `image-match`        

1. Run a NATS subject subscriber in a separate terminal,

        go get github.com/nats-io/nats.go/
        
        cd examples/nats-sub
        
        go run main.go -s localhost:4222 image-match

1. Now, its time to send a HTTP request to parameterize the notebook that add noise to original Argo logo and execute the image processing pipeline.

        curl -d '{"filterA":"15", "filterB": "15", "sVSp": "0.003", "amount": "0.010"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. List argo workflows to list the `noise-processor-` and `matcher-workflow-` 

        argo list

1. Check the noisy image in bucket `output`.

1. As soon as the `matcher-workflow-` completes, you will see a message on NATS subject

        `[#1] Received on [image-match]: 'FAILURE: 0.4500723255991941'`

1. Lets change the parameters for the curl request to get >80% match,

        curl -d '{"filterA":"5", "filterB": "5", "sVSp": "0.008", "amount": "0.0008"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
        
1. Still only 51% match,

        [#2] Received on [image-match]: 'FAILURE: 0.519408012194793'

1. Lets reduce the amount of noise to 0.0008,

        curl -d '{"filterA":"5", "filterB": "5", "sVSp": "0.008", "amount": "0.0008"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. You will see a success message,

        [#4] Received on [image-match]: 'SUCCESS: 0.9157008012270712'

1. This was a simple image processing pipeline using Argo Events. You can easily set up CI pipelines, Machine Learning pipelines etc,
   using Argo Events and Argo Workflows.
 
