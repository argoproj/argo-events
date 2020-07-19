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

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/resource.yaml

2. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/gateways/resource.yaml

3. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/resource.yaml

4. The event source we created in step 1 contains configuration which makes the gateway listen to 
   Argo workflows marked with label `app: my-workflow`.

5. Lets create a workflow called `my-workflow` with label `app: my-workflow`
   
        apiVersion: argoproj.io/v1alpha1
        kind: Workflow
        metadata:
          name: my-workflow
          labels:
            app: my-workflow
        spec:
          entrypoint: whalesay
          templates:
          - name: whalesay
            container:
              image: docker/whalesay:latest
              command: [cowsay]
              args: ["hello world"]

6. Once the `my-workflow` is created, the sensor will trigger the workflow. Run `argo list` to list the triggered workflow.

## List Options

The Resource Event-Source allows to configure the list options through `labels` and `field` selectors for setting up a watch on objects.

In the example above, we had set up the list option as follows,

        filter:
        # labels and filters are meant to provide K8s API options to filter the object list that are being watched.
        # Please read https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#api for more details.
        
            # labels provide listing options to K8s API to watch objects
            labels:
              - key: app
                # Supported operations like ==, !=, etc.
                # Defaults to ==.
                # Refer https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors for more info.
                # optional.
                operation: "=="
                value: my-workflow


The `key-operation-value` items under the `filter -> labels` are used by the gateway to filter the objects
that are eligible for the watch. So, in the present case, the gateway will set up a watch for those
objects who have label "app: my-workflow". You can add more `key-operation-value` items to the list as per your use-case.   

Similarly, you can pass `field` selectors to the watch list options, e.g.,

      filter:
        # labels and filters are meant to provide K8s API options to filter the object list that are being watched.
        # Please read https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#api for more details.

        # fields provide listing options to K8s API to watch objects
        fields:
          - key: metadata.name
            # Supported operations like ==, !=, <=, >= etc.
            # Defaults to ==.
            # Refer https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/ for more info.
            # optional.
            operation: ==
            value: my-workflow
 

**Note:** The `label` and `fields` under `filter` are used at the time of setting up the watch by the gateway. If you want to filter the objects
based on the `annotations` or some other fields, use the `Data Filters` available in sensor.

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
