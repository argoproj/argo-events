# File

The gateway watches changes to a file within specified directory.

## Where the directory should be?
The directory can be in the pod's own filesystem or you can mount a persistent volume and refer to a directory.
Make sure that the directory exists before you create the gateway configmap.

## Setup
1. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/file.yaml).

2. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/file.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/file.yaml).

## Trigger Workflow

Exec into the gateway pod and go to the directory specified in event source and create a file. That should generate an event causing sensor to trigger a workflow.

## How to listen to notifications from different directories
Simply edit the event source configmap and add new entry that contains the configuration required to listen to file within different directory and save
the configmap. The gateway will start listening to file notifications from new directory as well.
