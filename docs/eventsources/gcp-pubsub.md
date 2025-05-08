# GCP PubSub

## Topic And Subscription ID

GCP PubSub event source can listen to a PubSub with given `topic`, or
`subscriptionID`. Here is the logic with different `topic` and `subscriptionID`
combination.

| Topic Provided/Existing | Sub ID Provided/Existing | Actions                                                               |
| ----------------------- | ------------------------ | --------------------------------------------------------------------- |
| Yes/Yes                 | Yes/Yes                  | Validate if given topic matches subscription's topic                  |
| Yes/Yes                 | Yes/No                   | Create a subscription with given ID                                   |
| Yes/Yes                 | No/-                     | Create or reuse subscription with auto generated subID               |
| Yes/No                  | Yes/No                   | Create a topic and a subscription with given subID                    |
| Yes/No                  | Yes/Yes                  | Invalid                                                               |
| Yes/No                  | No/-                     | Create a topic, create or reuse subscription w/ auto generated subID |
| No/-                    | Yes/Yes                  | OK                                                                    |
| No/-                    | Yes/No                   | Invalid                                                               |

## Workload Identity

If you have configured
[Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
and want to use it for a PubSub EventSource, leave `credentialSecret` nil.

Full spec is available [here](../APIs.md#argoproj.io/v1alpha1.PubSubEventSource).

See a PubSub EventSource
[example](https://github.com/argoproj/argo-events/tree/stable/examples/event-sources/gcp-pubsub.yaml).

## Running With PubSub Emulator

You can point this event source at the
[PubSub Emulator](https://cloud.google.com/pubsub/docs/emulator) by
configuring the `PUBSUB_EMULATOR_HOST` environment variable for the event
source pod. This can be configured on the `EventSource` resource under the
`spec.template.container.env` key. This option is also documented in the
PubSub EventSource
[example](https://github.com/argoproj/argo-events/tree/stable/examples/event-sources/gcp-pubsub.yaml).
