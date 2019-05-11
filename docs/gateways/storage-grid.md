# StorageGrid

The gateway listens to bucket notifications from storage grid.

Note: The gateway does not register the webhook endpoint on storage grid. You need to do it manually. This is mainly because limitations of storage grid api.
The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from StorageGrid.

## How to get the URL for the service?
Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

1. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/storage-grid.yaml).

2. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/storage-grid.yaml).

3. Deployy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/storage-grid.yaml).

4. Configure notifications

       Go to your tenant page on StorageGRID
       Create an endpoint with the following values, and click save
          
          Display Name: S3 Notifications
          URI: <route-url-you-created>
          URN: urn:mytext:sns:us-east::my_topic
          Access Key: <your-access-key>
          Secret Key: <your-secret-key>
          Certificate Validation: <Do not verify>
        
       Go to the bucket for which you want to configure notifications.
          Enter the following XML string, and click save
         
      
          <NotificationConfiguration>
              <TopicConfiguration>
                  <Id>Object-Event</Id>
                  <Topic>urn:mytext:sns:us-east::my_topic</Topic>
                  <Event>s3:ObjectCreated:*</Event>
                  <Event>s3:ObjectRemoved:*</Event>
              </TopicConfiguration>
          </NotificationConfiguration>


## Trigger Workflow
Drop a file into the bucket for which you configured the notifications and watch Argo workflow being triggered.
