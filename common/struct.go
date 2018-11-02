package common

import (
	"github.com/minio/minio-go"
	corev1 "k8s.io/api/core/v1"
	"github.com/argoproj/argo-events/gateways"
)

// S3Artifact contains information about an artifact in S3
// +k8s:openapi-gen=true
type S3Artifact struct {
	S3Bucket `json:",inline" protobuf:"bytes,4,opt,name=s3Bucket"`
	Key      string                      `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
	Event    minio.NotificationEventType `json:"event,omitempty" protobuf:"bytes,2,opt,name=event"`
	Filter   *S3Filter                   `json:"filter,omitempty" protobuf:"bytes,3,opt,name=filter"`
}

// S3Bucket contains information for an S3 Bucket
type S3Bucket struct {
	Endpoint  string                   `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`
	Bucket    string                   `json:"bucket,omitempty" protobuf:"bytes,2,opt,name=bucket"`
	Region    string                   `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
	Insecure  bool                     `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`
	// +k8s:openapi-gen=false
	AccessKey corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,5,opt,name=accessKey"`
	// +k8s:openapi-gen=false
	SecretKey corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,6,opt,name=secretKey"`
}

// S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects
type S3Filter struct {
	Prefix string `json:"prefix" protobuf:"bytes,1,opt,name=prefix"`
	Suffix string `json:"suffix" protobuf:"bytes,2,opt,name=suffix"`
}

type DefaultConfigExecutor struct {
	StartChan chan struct{}
	StopChan chan struct{}
	DataCh chan []byte
	DoneCh chan struct{}
	ErrChan chan error
	GatewayConfig *gateways.GatewayConfig
}
