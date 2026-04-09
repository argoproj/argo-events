# Kafka

Kafka event-source listens to messages on topics and helps the sensor trigger workloads.

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
              "topic": "kafka_topic",
              "partition": "partition_number",
              "body": "message_body",
              "timestamp": "timestamp_of_the_message"
            }
        }

## Specification

Kafka event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.KafkaEventSource).

## Setup

1.  Make sure to set up the Kafka cluster in Kubernetes if you don't already have one. You can refer to <https://github.com/Yolean/kubernetes-kafka> for installation instructions.

1.  Create the event source by running the following command. Make sure to update the appropriate fields.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/kafka.yaml

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/kafka.yaml

1.  Send message by using Kafka client. More info on how to send message at <https://kafka.apache.org/quickstart>.

1.  Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).

## AWS MSK with IAM Authentication (IRSA)

When connecting to Amazon MSK with IAM access control enabled, use the `awsMskIamAuth` field instead of `sasl` or `tls`.
TLS is enabled automatically. The AWS credential chain is used, so the simplest way to grant access on EKS is via
[IRSA (IAM Roles for Service Accounts)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html).

### IAM policy for the role

The IAM role attached to the pod service account needs at minimum:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "kafka-cluster:Connect",
      "Resource": "arn:aws:kafka:<region>:<account-id>:cluster/<cluster-name>/<cluster-id>"
    },
    {
      "Effect": "Allow",
      "Action": [
        "kafka-cluster:DescribeTopic",
        "kafka-cluster:ReadData"
      ],
      "Resource": "arn:aws:kafka:<region>:<account-id>:topic/<cluster-name>/<cluster-id>/<topic-name>"
    },
    {
      "Effect": "Allow",
      "Action": "kafka-cluster:AlterGroup",
      "Resource": "arn:aws:kafka:<region>:<account-id>:group/<cluster-name>/<cluster-id>/<consumer-group-name>"
    }
  ]
}
```

### EventSource spec

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: kafka-msk
spec:
  kafka:
    msk-source:
      # Use port 9098 for IAM authentication (not 9092)
      url: b-1.my-cluster.abc123.c2.kafka.us-east-1.amazonaws.com:9098
      topic: my-topic
      consumerGroup:
        groupName: my-consumer-group
      awsMskIamAuth:
        region: us-east-1
```

### Annotate the service account

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: argo-events-sa
  namespace: argo-events
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::<account-id>:role/<role-name>
```

The `awsMskIamAuth` block uses the AWS SDK v2 default credential chain, so it also works with EC2 instance profiles
and static environment credentials (`AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `AWS_SESSION_TOKEN`) without
any additional configuration.
