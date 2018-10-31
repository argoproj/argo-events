package main

import (
	"github.com/minio/minio-go"
	corev1 "k8s.io/api/core/v1"
)

// S3Artifact contains information about an artifact in S3
type s3Artifact struct {
	// S3EventConfig contains configuration for bucket notification
	S3EventConfig *S3EventConfig `json:"s3EventConfig"`

	// Mode of operation for s3 client
	Insecure bool `json:"insecure,omitempty"`

	// AccessKey
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty"`

	// SecretKey
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty"`
}

// S3EventConfig contains configuration for bucket notification
type S3EventConfig struct {
	Endpoint string                      `json:"endpoint,omitempty"`
	Bucket   string                      `json:"bucket,omitempty"`
	Region   string                      `json:"region,omitempty"`
	Event    minio.NotificationEventType `json:"event,omitempty"`
	Filter   S3Filter                    `json:"filter,omitempty"`
}

// S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects
type S3Filter struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

