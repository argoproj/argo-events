# File

File gateway listens to file system events and helps sensor trigger workloads.


<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/file-setup.png?raw=true" alt="File Setup"/>
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
                "name": "Relative path to the file or directory",
                "op": "File operation that triggered the event" // Create, Write, Remove, Rename, Chmod
            }
        }


<br/>

## Setup

1. Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/file.yaml

2. The event source has configuration to listen to file system events for `test-data` directory and file called `x.txt`.

3. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/gateways/file.yaml

4. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/file.yaml

5. Make sure there are no errors in either gateway or sensor pod.

6. Log into the gateway pod by running following command,

        kubectl -n argo-events exec -it <file-gateway-pod-name> -c file-events -- /bin/bash

6. Lets create a file called `x.txt` under `test-data` directory in gateway pod.
 
        cd test-data
        cat <<EOF > x.txt
        hello
        EOF

8. Once you create file `x.txt`, the sensor will trigger argo workflow.  Run `argo list` to find the workflow. 

9. For real-world use cases, you should use PersistentVolume and PersistentVolumeClaim.
                                                                  
## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/). 
