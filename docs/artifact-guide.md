# Artifact Guide
`Sensors` supports S3 artifact repositories. To take advantage of these artifact signals, you can use this guide to get started with installing artifact services.

### Prerequisites
Up & Running kubernetes instance

## Minio
[Minio](https://www.minio.io/) is an distributed object storage server. 

### Install
```
$ brew install kubernetes-helm # mac
$ helm init
$ helm repo add stable https://kubernetes-charts.storage.googleapis.com
$ helm repo update
$ helm install stable/minio --name artifacts --set service.type=LoadBalancer
$ helm upgrade -f docs/minio-config.yaml stable/minio
```

Note: When minio is installed via Helm, it uses the following hard-wired default credentials and creates a kubernetes secret with a hash of these keys. Signal specifications rely on kubernetes secrets instead of the raw keys. 
- AccessKey: AKIAIOSFODNN7EXAMPLE
- SecretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
