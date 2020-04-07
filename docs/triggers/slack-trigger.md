# Slack Trigger

The Slack trigger is used to send a custom message to a desired Slack channel in a Slack workspace. The intended use is for notifications for a build pipeline, but can be used for any notification scenario. 

## Prerequisite
Have a Slack workspace setup you wish to send a message to. Set up the webhook gateway and event source. The K8s manifests are available under `examples`.

1. Set up a simple webhook gateway.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml

2. Create a webhook event source.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml

3. Set up port-forwarding to expose the http server. We will
   use port-forwarding here.
   
        kubectl port-forward -n argo-events webhook-gateway-YOUR-GATEWAY-POD 12000:12000

## Create a Slack App
We need to create a Slack App which will send messages to your Slack Workspace. We will add OAuth Permissions and add the OAuth token to the k8s cluster via a secret.

1. Create a Slack app by clicking `Create New App` at the [Slack API Page](https://api.slack.com/apps). Name your app and choose your intended Slack Workspace

2. Navigate to your app, then to `Features > OAuth & Permissions`

3. Scroll down to `Scopes` and add the scopes `channels:join`, and `chat:write`

4. Scroll to the top of the `OAuth & Permissions` page and click `Install App to Workspace` and follow the install Wizard

5. You should land back on the `OAuth & Permissions` page. Copy your app's OAuth Access Token. This will allow the trigger to act on behalf of your newly created Slack app.

6. Encode your OAuth token in base64. This can done easily with the command line

        echo -n "YOUR-OAUTH-TOKEN" | base64

7. Create a kubernetes secret file `slack-secret.yaml` with your OAuth token in the following format

        apiVersion: v1
        kind: Secret
        metadata:
          name: slack-secret
        data:
          token: YOUR-BASE64-ENCODED-OAUTH-TOKEN

12. Apply the kubernetes secret

        kubectl -n argo-events apply -f slack-secret.yaml

## Slack Trigger
We will set up a basic slack trigger and send a default message, and then a dynamic custom message. 

1. Create a sensor with Slack trigger. We will discuss the trigger details in the following sections.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/slack-trigger.yaml

2. Send a http request to the newly setup gateway to fire the Slack trigger. 

        curl -d '{"text":"Hello, World!"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
        
      **Note**: The default slack-trigger will send the message "hello world" to the #general channel. You may change the default message and channel in slack-trigger.yaml under triggers.slack.channel and triggers.slack.message.
3. Alternatively, you can dynamically determine the channel and message based on parameterization of your event. 

        curl -d '{"channel":"random","message":"test message"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

4. Great! But how did the sensor use the event to customize the message and channel from the http request? We will see that in next section.

## Parameterization
The slack trigger parameters have the following structure,

        parameters:
          - src:
              dependencyName: test-dep
              dataKey: body.channel
            dest: slack.channel
          - src:
              dependencyName: test-dep
              contextKey: body.message
            dest: slack.message

The `src` is the source of event. It contains,

  1. `dependencyName`: name of the event dependency to extract the event from.
  2. `dataKey`: to extract a particular key-value from event's data.
  3. `contextKey`: to extract a particular key-value from event' context.

The `dest` is the destination key within the result payload.

So, the above trigger paramters will generate a request payload as,

        {
            "channel": "channel_to_send_message",
            "message": "message_to_send_to_channel"
        }


**_Note_**: If you define both the `contextKey` and `dataKey` within a paramter item, then
the `dataKey` takes the precedence.

You can create any paramater structure you want. To get more info on how to 
generate complex event payloads, take a look at [this library](https://github.com/tidwall/sjson).

The complete specification of Slack trigger is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#slacktrigger).
