# Alibabacloud MNS/SMQ

MNS event-source subscribes to Alibaba Cloud MNS/SMQ topics, listen events and helps sensor trigger the workflows.

## Event Structure

```js
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
        "messageId": "mns message id",
        "body": "body refers to the alibaba mns notification data",
    }
}
```

## Setup

1. Create a topic called `test-event-topic` using Alibaba Cloud MNS/SMQ console, it will generate a public http endpoint.

2. Fetch your access and secret key for Alibaba Cloud account.

3. Create a secret called `mns-secret` as follows with command:

```bash
kubectl create secret generic mns-secret\
    --from-literal=accesskey=*** \
    --from-literal=secretkey=***
```

4. create event-source.yaml

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: ali-mns
spec:
  mns:
    example:
      jsonBody: true
      accessKey:
        key: accesskey
        name: mns-secret
      secretKey:
        key: secretkey
        name: mns-secret
      waitTimeSeconds: 20
      endpoint: http://165***368.mns.<region>.aliyuncs.com # the endpoint generated in step 1
```

5. The event-source for Alibabacloud MNS/SMQ creates a pod. Use `kubectl get pod` check if that works well.
