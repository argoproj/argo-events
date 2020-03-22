# Resource

Resource gateway watches change notifications for K8s object and helps sensor trigger the workloads.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/resource-setup.png?raw=true" alt="Resource Setup"/>
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
              "type": "type_of_the_event", // ADD, UPDATE or DELETE
              "body": "resource_body", // JSON format
              "group": "resource_group_name",
              "version": "resource_version_name",
              "resource": "resource_name"
            }
        }

<br/>

## Setup
1. Create the event source by running the following command. 

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/resource.yaml

2. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/resource.yaml

3. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/resource.yaml

4. The event source we created in step 1 contains configuration which makes the gateway listen to 
   Argo workflows marked with label and `name: my-workflow`.

5. Lets create a workflow called `my-workflow`
   
        apiVersion: argoproj.io/v1alpha1
        kind: Workflow
        metadata:
          name: my-workflow
          labels:
            name: my-workflow
        spec:
          entrypoint: whalesay
          templates:
          - name: whalesay
            container:
              image: docker/whalesay:latest
              command: [cowsay]
              args: ["hello world"]

6. Once the `my-workflow` is created, the sensor will trigger the workflow. Run `argo list` to list the triggered workflow.

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
