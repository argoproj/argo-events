# GCP PubSub

## Topic And Subscription ID

GCP PubSub event source can listen to a PubSub with given `topic`, or
`subscriptionID`. Here is the logic with different `topic` and `subscriptionID`
combination.

| Topic Provided/Existing | Sub ID Provided/Existing | Actions                                                               |
| ----------------------- | ------------------------ | --------------------------------------------------------------------- |
| Yes/Yes                 | Yes/Yes                  | Validate if given topic matches subsciption's topic                   |
| Yes/Yes                 | Yes/No                   | Create a subscription with given ID                                   |
| Yes/Yes                 | No/-                     | Create or re-use subscription with auto generated subID               |
| Yes/No                  | Yes/No                   | Create a topic and a subscription with given subID                    |
| Yes/No                  | Yes/Yes                  | Invalid                                                               |
| Yes/No                  | No/-                     | Create a topic, create or re-use subscription w/ auto generated subID |
| No/-                    | Yes/Yes                  | OK                                                                    |
| No/-                    | Yes/No                   | Invalid                                                               |

## Workload Identity

If you have configured
[Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
and want to use it for a PubSub EventSource, leave `credentialSecret` nil.

Full spec is available [here](../../api/event-source.md#pubsubeventsource).

See a PubSub EventSource
[example](../../examples/event-sources/gcp-pubsub.yaml).
