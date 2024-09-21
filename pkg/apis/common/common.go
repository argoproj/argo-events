/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	corev1 "k8s.io/api/core/v1"
)

// EventSourceType is the type of event source
type EventSourceType string

// possible event source types
var (
	MinioEvent           EventSourceType = "minio"
	CalendarEvent        EventSourceType = "calendar"
	FileEvent            EventSourceType = "file"
	SFTPEvent            EventSourceType = "sftp"
	ResourceEvent        EventSourceType = "resource"
	WebhookEvent         EventSourceType = "webhook"
	AMQPEvent            EventSourceType = "amqp"
	KafkaEvent           EventSourceType = "kafka"
	MQTTEvent            EventSourceType = "mqtt"
	NATSEvent            EventSourceType = "nats"
	SNSEvent             EventSourceType = "sns"
	SQSEvent             EventSourceType = "sqs"
	PubSubEvent          EventSourceType = "pubsub"
	GerritEvent          EventSourceType = "gerrit"
	GithubEvent          EventSourceType = "github"
	GitlabEvent          EventSourceType = "gitlab"
	HDFSEvent            EventSourceType = "hdfs"
	SlackEvent           EventSourceType = "slack"
	StorageGridEvent     EventSourceType = "storagegrid"
	AzureEventsHub       EventSourceType = "azureEventsHub"
	AzureQueueStorage    EventSourceType = "azureQueueStorage"
	AzureServiceBus      EventSourceType = "azureServiceBus"
	StripeEvent          EventSourceType = "stripe"
	EmitterEvent         EventSourceType = "emitter"
	RedisEvent           EventSourceType = "redis"
	RedisStreamEvent     EventSourceType = "redisStream"
	NSQEvent             EventSourceType = "nsq"
	PulsarEvent          EventSourceType = "pulsar"
	GenericEvent         EventSourceType = "generic"
	BitbucketServerEvent EventSourceType = "bitbucketserver"
	BitbucketEvent       EventSourceType = "bitbucket"
)

var (
	// RecreateStrategyEventSources refers to the list of event source types
	// that need to use Recreate strategy for its Deployment
	RecreateStrategyEventSources = []EventSourceType{
		AMQPEvent,
		CalendarEvent,
		KafkaEvent,
		PubSubEvent,
		AzureEventsHub,
		AzureServiceBus,
		NATSEvent,
		MQTTEvent,
		MinioEvent,
		EmitterEvent,
		NSQEvent,
		PulsarEvent,
		RedisEvent,
		RedisStreamEvent,
		ResourceEvent,
		HDFSEvent,
		FileEvent,
		SFTPEvent,
		GenericEvent,
	}
)

// TriggerType is the type of trigger
type TriggerType string

// possible trigger types
var (
	OpenWhiskTrigger       TriggerType = "OpenWhisk"
	ArgoWorkflowTrigger    TriggerType = "ArgoWorkflow"
	LambdaTrigger          TriggerType = "Lambda"
	CustomTrigger          TriggerType = "Custom"
	HTTPTrigger            TriggerType = "HTTP"
	KafkaTrigger           TriggerType = "Kafka"
	PulsarTrigger          TriggerType = "Pulsar"
	LogTrigger             TriggerType = "Log"
	NATSTrigger            TriggerType = "NATS"
	SlackTrigger           TriggerType = "Slack"
	K8sTrigger             TriggerType = "Kubernetes"
	AzureEventHubsTrigger  TriggerType = "AzureEventHubs"
	AzureServiceBusTrigger TriggerType = "AzureServiceBus"
	EmailTrigger           TriggerType = "Email"
)

// EventBusType is the type of event bus
type EventBusType string

// possible event bus types
var (
	EventBusNATS      EventBusType = "nats"
	EventBusJetStream EventBusType = "jetstream"
	EventBusKafka     EventBusType = "kafka"
)

// SecureHeader refers to HTTP Headers with auth tokens as values
type SecureHeader struct {
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Values can be read from either secrets or configmaps
	ValueFrom *ValueFromSource `json:"valueFrom,omitempty" protobuf:"bytes,2,opt,name=valueFrom"`
}

// ValueFromSource allows you to reference keys from either a Configmap or Secret
type ValueFromSource struct {
	SecretKeyRef    *corev1.SecretKeySelector    `json:"secretKeyRef,omitempty" protobuf:"bytes,1,opt,name=secretKeyRef"`
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty" protobuf:"bytes,2,opt,name=configMapKeyRef"`
}

// SchemaRegistryConfig refers to configuration for a client
type SchemaRegistryConfig struct {
	// Schema Registry URL.
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Schema ID
	SchemaID int32 `json:"schemaId" protobuf:"varint,2,name=schemaId"`
	// +optional
	// SchemaRegistry - basic authentication
	Auth BasicAuth `json:"auth,omitempty" protobuf:"bytes,3,opt,name=auth"`
}
