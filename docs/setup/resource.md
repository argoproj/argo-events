# Resource Gateway & Sensor

Resource gateway watches change notifications for K8s object and helps sensor trigger the workloads.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/docs-gateway-setup/docs/assets/resource-setup.png?raw=true" alt="Resource Setup"/>
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
              "type": "type_of_the_event", // CREATE, UPDATE or DELETE
              "body": "resource_body",
              "group": "resource_group_name",
              "version": "resource_version_name",
              "resource": "resource_name"
            }
        }

<br/>

## Setup
1. Create the event source by running the following command. 

   **The event source has multiple configurations that you may not be interested in. Make sure to update the appropriate fields.**

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/resource.yaml

2. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/resource.yaml

3. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/resource.yaml
        
4. If you create an argo workflow called `my-workflow`, once it transitions into `success` state, the sensor will trigger another argo workflow.

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).

