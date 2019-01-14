# Artifact Guide
This is a guide for help in utilizing artifacts within Argo Events. Sensors use artifacts for Resource Object store for use in `Resource` triggers

## Inline
Inlined artifacts are included directly within the sensor resource and decoded as a string.

## S3
Amazon Simple Storage Service (S3) is a block/file/object store for the internet. The standardized [API](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html) allows storage and retrieval of data at any time from anywhere on the web. There are a number of S3 providers that include, but are not limited to:
* [Amazon S3](https://aws.amazon.com/s3/?nc2=h_m1)
* [Minio](https://minio.io/)
* [NetApp](https://www.netapp.com/us/products/data-management-software/object-storage-grid-sds.aspx)
* [CEPH](http://docs.ceph.com/docs/master/radosgw/s3/)
* [Rook](https://rook.io/)

### Minio
Argo Events uses the [minio-go](https://github.com/minio/minio-go) client for access to any Amazon S3 compatible object store. [Minio](https://www.minio.io/) is an distributed object storage server. Follow the Minio [Bucket Notification Guide](https://docs.minio.io/docs/minio-bucket-notification-guide) for help with configuring your minio server to listen and monitor for bucket event notifications. Note that you will need to setup a supported message queue for configuring your notification targets (i.e. NATS, WebHooks, Kafka, etc.). 

#### Installation on Kubernetes
The [Minio Deployment Quickstart Guide](https://docs.minio.io/docs/minio-deployment-quickstart-guide.html) is useful for help in getting Minio up & running on your orchestration platform. We've also outlined additional steps below to use Minio for signal notifications and as an object store for trigger resources.

1. Install the Helm chart
```
$ helm init
...
$ helm install stable/minio --name artifacts --set service.type=LoadBalancer
...

$ #Verify that the minio pod, the minio service and minio secret are present
$ kubectl get all -n default -l app=minio

NAME                                   READY     STATUS    RESTARTS   AGE
pod/artifacts-minio-85547b6bd9-bhtt8   1/1       Running   0          21m

NAME                      TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
service/artifacts-minio   ClusterIP   None         <none>        9000/TCP   21m

NAME                              DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/artifacts-minio   1         1         1            1           21m

NAME                                         DESIRED   CURRENT   READY     AGE
replicaset.apps/artifacts-minio-85547b6bd9   1         1         1         21m
```

2. Create a bucket in Minio and upload the hello-world.yaml into that bucket.
Download the hello-world.yaml from https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
```
$ kubectl port-forward `kubectl get pod -l app=minio -o name` 9000:9000
```
Open the browser at http://localhost:9000
Create a new bucket called 'workflows'.
Upload the hello-world.yaml into that bucket


#### Enabling bucket notifications
Once the Minio server is configured with a notification target and you have restarted the server to put the changes into effect, you now need to explicitely enable event notifications for a specified bucket. Enabling these notifications are out of scope of Argo Events since bucket notifications are a construct within Minio that exists at the `bucket` level. To avoid multiple sensors on the same S3 bucket conflicting with each other, creating, updating, and deleting Minio bucket notifications should be delegated to a separate process with knowledge of all notification targets including those outside of the Argo Events.
```
$ k edit configmap artifacts-minio
$ k delete pod artifacts-minio
```

## File (future enhancement)
This will enable access to file artifacts via a filesystem mounted as a [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) within the `sensor-controller` pod. 

## URL (future enhancement)
This will enable access to web artifacts via RESTful API.