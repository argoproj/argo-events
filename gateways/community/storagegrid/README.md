<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/netapp.png?raw=true" alt="NetApp"/>
</p>

<br/>

# StorageGrid

The gateway listens to bucket notifications from storage grid.

### How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// Webhook
Hook *common.Webhook `json:"hook"`

// Events are s3 bucket notification events.
// For more information on s3 notifications, follow https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations
// Note that storage grid notifications do not contain `s3:`
Events []string `json:"events,omitempty"`

// Filter on object key which caused the notification.
Filter *Filter `json:"filter,omitempty"`
```

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: storage-grid-gateway-configmap
data:
  foo: |-
    hook:
      port: "8080"
      endpoint: "/"
    events:
      - "ObjectCreated:Put"
```

Note: The gateway does not register the webhook endpoint on storage grid. You need to do it manually. 
This is mainly because limitations of storage grid api.

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from StorageGrid.

**How to get the URL for the service?**

Depending upon the Kubernetes provider, you can create the Ingress or Route. 


## Setup

**1. Install Gateway**
```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/storage-grid.yaml
```

Make sure gateway pod and service is running

**2. Install Gateway Configmap**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/storage-grid-gateway-configmap.yaml
```

**3. Install Sensor**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/storage-grid.yaml
```

Make sure sensor pod is created.

**4. Configure notifications**

   * Go to your tenant page on StorageGRID
   * Create an endpoint with the following values, and click save
      ```
      Display Name: S3 Notifications
      URI: <route-url-you-created>
      URN: urn:mytext:sns:us-east::my_topic
      Access Key: <your-access-key>
      Secret Key: <your-secret-key>
      Certificate Validation: <Do not verify>
      ```
    
   * Go to the bucket for which you want to configure notifications.
      Enter the following XML string, and click save
     
      ```xml
      <NotificationConfiguration>
          <TopicConfiguration>
              <Id>Object-Event</Id>
              <Topic>urn:mytext:sns:us-east::my_topic</Topic>
              <Event>s3:ObjectCreated:*</Event>
              <Event>s3:ObjectRemoved:*</Event>
          </TopicConfiguration>
      </NotificationConfiguration>
      ```

**5. Trigger Workflow**

Drop a file into the bucket for which you configured the notifications and watch Argo workflow being triggered.
