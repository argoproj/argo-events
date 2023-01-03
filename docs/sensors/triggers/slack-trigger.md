# Slack Trigger

The Slack trigger is used to send a custom message to a desired Slack channel in a Slack workspace. The intended use is for notifications for a build pipeline, but can be used for any notification scenario.

## Prerequisite

1. Deploy the eventbus in the namespace.

1. Make sure to have a Slack workspace setup you wish to send a message to.

2. Create a webhook event-source.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

3. Set up port-forwarding to expose the http server. We will
   use port-forwarding here.

        kubectl port-forward -n argo-events <event-source-pod-name> 12000:12000

## Create a Slack App

We need to create a Slack App which will send messages to your Slack Workspace. We will add OAuth Permissions and add the OAuth token to the k8s cluster via a secret.

1. Create a Slack app by clicking `Create New App` at the [Slack API Page](https://api.slack.com/apps). Name your app and choose your intended Slack Workspace.

2. Navigate to your app, then to `Features > OAuth & Permissions`.

3. Scroll down to `Scopes` and add the scopes `channels:join`, `channels:read`, `groups:read` and `chat:write` to the _Bot Token Scopes_.

4. Scroll to the top of the `OAuth & Permissions` page and click `Install App to Workspace` and follow the install Wizard.

5. You should land back on the `OAuth & Permissions` page. Copy your app's OAuth Access Token. This will allow the trigger to act on behalf of your newly created Slack app.

6. Create a kubernetes secret with the OAuth token in your cluster.

        kubectl create secret generic slack-secret --from-literal=token=$SLACK_OAUTH_TOKEN

## Slack Trigger

We will set up a basic slack trigger and send a default message, and then a dynamic custom message.

1. Create a sensor with Slack trigger. We will discuss the trigger details in the following sections.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/slack-trigger.yaml

2. Send a http request to the event-source-pod to fire the Slack trigger.

        curl -d '{"text":"Hello, World!"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

      **Note**: The default slack-trigger will send the message "hello world" to the #general channel. You may change the default message and channel in slack-trigger.yaml under triggers.slack.channel and triggers.slack.message.

3. Alternatively, you can dynamically determine the channel and message based on parameterization of your event.

        curl -d '{"channel":"random","message":"test message"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

4. Great! But, how did the sensor use the event to customize the message and channel from the http request? We will see that in next section.

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

So, the above trigger parameters will generate a request payload as,

        {
            "channel": "channel_to_send_message",
            "message": "message_to_send_to_channel"
        }

**_Note_**: If you define both the `contextKey` and `dataKey` within a parameter item, then
the `dataKey` takes the precedence.

You can create any parameter structure you want. To get more info on how to
generate complex event payloads, take a look at [this library](https://github.com/tidwall/sjson).

## Other Capabilities

#### Configuring the sender of the Slack message: 

         - template:
              name: slack-trigger
              slack:
                sender:
                  username: "Cool Robot"
                  icon: ":robot_face:" # emoji or url, e.g. https://example.com/image.png

#### Sending messages to Slack threads:

         - template:
              name: slack-trigger
              slack:
                thread:
                  messageAggregationKey: "abcdefg" # aggregate message by some key to send them to the same Slack thread
                  broadcastMessageToChannel: true # also broadcast the message from the thread to the channel

#### Sending attachments using [Slack Attachments API](https://api.slack.com/reference/messaging/attachments):

         - template:
              name: slack-trigger
              slack:
                message: "hello world!"
                attachments: |
                  [{
                    "title": "Attachment1!",
                    "title_link": "https://argoproj.github.io/argo-events/sensors/triggers/slack-trigger/",
                    "color": "#18be52",
                    "fields": [{
                      "title": "Hello1",
                      "value": "Hello World1",
                      "short": true
                    }, {
                      "title": "Hello2",
                      "value": "Hello World2",
                      "short": true
                    }]
                  }, {
                    "title": "Attachment2!",
                    "title_link": "https://argoproj.github.io/argo-events/sensors/triggers/slack-trigger/",
                    "color": "#18be52",
                    "fields": [{
                      "title": "Hello1",
                      "value": "Hello World1",
                      "short": true
                    }, {
                      "title": "Hello2",
                      "value": "Hello World2",
                      "short": true
                    }]
                  }]

#### Sending blocks using [Slack Blocks API](https://api.slack.com/reference/block-kit/blocks):

         - template:
              name: slack-trigger
              slack:
                blocks: |
                  [{
                    "type": "actions",
                    "block_id": "actionblock789",
                    "elements": [{
                        "type": "datepicker",
                        "action_id": "datepicker123",
                        "initial_date": "1990-04-28",
                        "placeholder": {
                          "type": "plain_text",
                          "text": "Select a date"
                        }
                      },
                      {
                        "type": "overflow",
                        "options": [{
                            "text": {
                              "type": "plain_text",
                              "text": "*this is plain_text text*"
                            },
                            "value": "value-0"
                          },
                          {
                            "text": {
                              "type": "plain_text",
                              "text": "*this is plain_text text*"
                            },
                            "value": "value-1"
                          },
                          {
                            "text": {
                              "type": "plain_text",
                              "text": "*this is plain_text text*"
                            },
                            "value": "value-2"
                          },
                          {
                            "text": {
                              "type": "plain_text",
                              "text": "*this is plain_text text*"
                            },
                            "value": "value-3"
                          },
                          {
                            "text": {
                              "type": "plain_text",
                              "text": "*this is plain_text text*"
                            },
                            "value": "value-4"
                          }
                        ],
                        "action_id": "overflow"
                      },
                      {
                        "type": "button",
                        "text": {
                          "type": "plain_text",
                          "text": "Click Me"
                        },
                        "value": "click_me_123",
                        "action_id": "button"
                      }
                    ]
                  }]

The complete specification of Slack trigger is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#slacktrigger).
