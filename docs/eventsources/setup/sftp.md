# SFTP

SFTP event-source polls an SFTP server to identify changes and helps sensor trigger workloads.

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
                "name": "Relative path to the file or directory",
                "op": "File operation that triggered the event" // Create, Remove
            }
        }

## Specification

SFTP event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.SFTPEventSource).

## Setup

1.  Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/sftp.yaml

1.  The event source has configuration to poll the sftp server every 10 seconds for `test-data` directory and file(s) called `x.txt`.

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/sftp.yaml

1.  Log into the event-source pod by running following command.

        kubectl -n argo-events exec -it <event-source-pod-name> -c sftp-events -- /bin/bash

1.  Create a file called `x.txt` under `test-data` directory on the SFTP server.

1.  Once you create file `x.txt`, the sensor will trigger argo workflow. Run `argo list` to find the workflow.

1.  For real-world use cases, you should use PersistentVolumeClaim.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
