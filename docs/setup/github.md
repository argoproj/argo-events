# GitHub

GitHub gateway programatically configures webhooks for projects on GitHub and helps sensor trigger the workloads upon events.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/github-setup.png?raw=true" alt="GitHub Setup"/>
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
                  	"body": "Body is the github event data",
                  	"headers": "Headers from the Gitlab event",
                }
            }

<br/>

## Setup

1. Create an API token if you don't have one. Follow [instructions](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line) to create a new GitHub API Token.
   Grant it the `repo_hook` permissions. 

2. Base64 encode your api token key,

        echo -n <api-token-key> | base64

3. Create a secret called `github-access`.

        apiVersion: v1
        kind: Secret
        metadata:
          name: github-access
        type: Opaque
        data:
          access: <base64-encoded-api-token-from-previous-step>

4. Deploy the secret into K8s cluster

        kubectl -n argo-events apply -f github-access.yaml

5. Create the gateway by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/github.yaml

6. Wait for gateway pod to get into the running state.

7. Create an Ingress or Openshift Route for the gateway service to that it can be reached from GitHub.
   You can find more information on Ingress or Route online.

8. Get the event source stored at https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/github.yaml

9. Change the `url` under `webhook` to your gateway service url created in a previous step. Make sure this url is reachable from GitHub.

8. Create the event source by running the following command.
   
        kubectl apply -n argo-events -f <event-source-file-updated-in-previous-step>

11. Go to `Webhooks` under your project settings on GitHub and verify the webhook is registered. You can also do the same by
    looking at the gateway pod logs.
    
12. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/github.yaml

13. Make a change to one of your project files and commit. It will trigger an argo workflow.

14. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/faq/).
