<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/ebdbdd4a2a8ce47a0fc6e9a6a63531be2c26148a/docs/assets/netapp.png?raw=true" alt="NetApp"/>
</p>

<br/>

# StorageGrid

The gateway listens to bucket notifications from storage grid.

### How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/community/slack/config.go#L38-L41).

Note: The gateway does not register the webhook endpoint on storage grid. You need to do it manually. 
This is mainly because limitations of storage grid api.

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from StorageGrid.

**How to get the URL for the service?**

Depending upon the Kubernetes provider, you can create the Ingress or Route. 


## Setup

**1. Install [Gateway](../../../examples/gateways/storage-grid.yaml)**

Make sure gateway pod and service is running

**2. Install [Gateway Configmap](../../../examples/gateways/storage-grid-gateway-configmap.yaml)**

**3. Install [Sensor](../../../examples/sensors/storage-grid.yaml)**

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
