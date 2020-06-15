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
package v1alpha1

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
)

// EventSource is the definition of a eventsource resource
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type EventSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Status            EventSourceStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	Spec              EventSourceSpec   `json:"spec" protobuf:"bytes,3,opt,name=spec"`
}

// EventSourceList is the list of eventsource resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EventSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	// +listType=eventsource
	Items []EventSource `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// EventSourceSpec refers to specification of event-source resource
type EventSourceSpec struct {
	// Minio event sources
	Minio map[string]apicommon.S3Artifact `json:"minio,omitempty" protobuf:"bytes,1,rep,name=minio"`
	// Calendar event sources
	Calendar map[string]CalendarEventSource `json:"calendar,omitempty" protobuf:"bytes,2,rep,name=calendar"`
	// File event sources
	File map[string]FileEventSource `json:"file,omitempty" protobuf:"bytes,3,rep,name=file"`
	// Resource event sources
	Resource map[string]ResourceEventSource `json:"resource,omitempty" protobuf:"bytes,4,rep,name=resource"`
	// Webhook event sources
	Webhook map[string]Context `json:"webhook,omitempty" protobuf:"bytes,5,rep,name=webhook"`
	// AMQP event sources
	AMQP map[string]AMQPEventSource `json:"amqp,omitempty" protobuf:"bytes,6,rep,name=amqp"`
	// Kafka event sources
	Kafka map[string]KafkaEventSource `json:"kafka,omitempty" protobuf:"bytes,7,rep,name=kafka"`
	// MQTT event sources
	MQTT map[string]MQTTEventSource `json:"mqtt,omitempty" protobuf:"bytes,8,rep,name=mqtt"`
	// NATS event sources
	NATS map[string]NATSEventsSource `json:"nats,omitempty" protobuf:"bytes,9,rep,name=nats"`
	// SNS event sources
	SNS map[string]SNSEventSource `json:"sns,omitempty" protobuf:"bytes,10,rep,name=sns"`
	// SQS event sources
	SQS map[string]SQSEventSource `json:"sqs,omitempty" protobuf:"bytes,11,rep,name=sqs"`
	// PubSub eevnt sources
	PubSub map[string]PubSubEventSource `json:"pubSub,omitempty" protobuf:"bytes,12,rep,name=pubSub"`
	// Github event sources
	Github map[string]GithubEventSource `json:"github,omitempty" protobuf:"bytes,13,rep,name=github"`
	// Gitlab event sources
	Gitlab map[string]GitlabEventSource `json:"gitlab,omitempty" protobuf:"bytes,14,rep,name=gitlab"`
	// HDFS event sources
	HDFS map[string]HDFSEventSource `json:"hdfs,omitempty" protobuf:"bytes,15,rep,name=hdfs"`
	// Slack event sources
	Slack map[string]SlackEventSource `json:"slack,omitempty" protobuf:"bytes,16,rep,name=slack"`
	// StorageGrid event sources
	StorageGrid map[string]StorageGridEventSource `json:"storageGrid,omitempty" protobuf:"bytes,17,rep,name=storageGrid"`
	// AzureEventsHub event sources
	AzureEventsHub map[string]AzureEventsHubEventSource `json:"azureEventsHub,omitempty" protobuf:"bytes,18,rep,name=azureEventsHub"`
	// Stripe event sources
	Stripe map[string]StripeEventSource `json:"stripe,omitempty" protobuf:"bytes,19,rep,name=stripe"`
	// Emitter event source
	Emitter map[string]EmitterEventSource `json:"emitter,omitempty" protobuf:"bytes,20,rep,name=emitter"`
	// Redis event source
	Redis map[string]RedisEventSource `json:"redis,omitempty" protobuf:"bytes,21,rep,name=redis"`
	// NSQ event source
	NSQ map[string]NSQEventSource `json:"nsq,omitempty" protobuf:"bytes,22,rep,name=nsq"`
	// Generic event source
	Generic map[string]GenericEventSource `json:"generic,omitempty" protobuf:"bytes,23,rep,name=generic"`
	// Type of the event source
	Type apicommon.EventSourceType `json:"type" protobuf:"bytes,24,opt,name=type,casttype=github.com/argoproj/argo-events/pkg/apis/common.EventSourceType"`
}

// CalendarEventSource describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type CalendarEventSource struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule" protobuf:"bytes,1,opt,name=schedule"`
	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval" protobuf:"bytes,2,opt,name=interval"`
	// ExclusionDates defines the list of DATE-TIME exceptions for recurring events.
	// +listType=string
	ExclusionDates []string `json:"exclusionDates,omitempty" protobuf:"bytes,3,rep,name=exclusionDates"`
	// Timezone in which to run the schedule
	// +optional
	Timezone string `json:"timezone,omitempty" protobuf:"bytes,4,opt,name=timezone"`
	// UserPayload will be sent to sensor as extra data once the event is triggered
	// +optional
	UserPayload json.RawMessage `json:"userPayload,omitempty" protobuf:"bytes,5,opt,name=userPayload,casttype=encoding/json.RawMessage"`
}

// FileEventSource describes an event-source for file related events.
type FileEventSource struct {
	// Type of file operations to watch
	// Refer https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go for more information
	EventType string `json:"eventType" protobuf:"bytes,1,opt,name=eventType"`
	// WatchPathConfig contains configuration about the file path to watch
	WatchPathConfig WatchPathConfig `json:"watchPathConfig" protobuf:"bytes,2,opt,name=watchPathConfig"`
	// Use polling instead of inotify
	Polling bool `json:"polling,omitempty" protobuf:"varint,3,opt,name=polling"`
}

// ResourceEventType is the type of event for the K8s resource mutation
type ResourceEventType string

// possible values of ResourceEventType
const (
	ADD    ResourceEventType = "ADD"
	UPDATE ResourceEventType = "UPDATE"
	DELETE ResourceEventType = "DELETE"
)

// ResourceEventSource refers to a event-source for K8s resource related events.
type ResourceEventSource struct {
	// Namespace where resource is deployed
	Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	// Filter is applied on the metadata of the resource
	// If you apply filter, then the internal event informer will only monitor objects that pass the filter.
	// +optional
	Filter *ResourceFilter `json:"filter,omitempty" protobuf:"bytes,2,opt,name=filter"`
	// Group of the resource
	metav1.GroupVersionResource `json:",inline" protobuf:"bytes,3,opt,name=groupVersionResource"`
	// EventTypes is the list of event type to watch.
	// Possible values are - ADD, UPDATE and DELETE.
	EventTypes []ResourceEventType `json:"eventTypes" protobuf:"bytes,4,rep,name=eventTypes,casttype=ResourceEventType"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource event objects
type ResourceFilter struct {
	// Prefix filter is applied on the resource name.
	// +optional
	Prefix string `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	// Labels provide listing options to K8s API to watch resource/s.
	// Refer https://kubernetes.io/docs/concepts/overview/working-with-objects/label-selectors/ for more info.
	// +optional
	Labels []Selector `json:"labels,omitempty" protobuf:"bytes,2,rep,name=labels"`
	// Fields provide listing options to K8s API to watch resource/s.
	// Refer https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/ for more info.
	// +optional
	Fields []Selector `json:"fields,omitempty" protobuf:"bytes,3,rep,name=fields"`
	// If resource is created before the specified time then the event is treated as valid.
	// +optional
	CreatedBy metav1.Time `json:"createdBy,omitempty" protobuf:"bytes,4,opt,name=createdBy"`
	// If the resource is created after the start time then the event is treated as valid.
	// +optional
	AfterStart bool `json:"afterStart,omitempty" protobuf:"varint,5,opt,name=afterStart"`
}

// Selector represents conditional operation to select K8s objects.
type Selector struct {
	// Key name
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// Supported operations like ==, !=, <=, >= etc.
	// Defaults to ==.
	// Refer https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors for more info.
	// +optional
	Operation string `json:"operation,omitempty" protobuf:"bytes,2,opt,name=operation"`
	// Value
	Value string `json:"value" protobuf:"bytes,3,opt,name=value"`
}

// AMQPEventSource refers to an event-source for AMQP stream events
type AMQPEventSource struct {
	// URL for rabbitmq service
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// ExchangeName is the exchange name
	// For more information, visit https://www.rabbitmq.com/tutorials/amqp-concepts.html
	ExchangeName string `json:"exchangeName" protobuf:"bytes,2,opt,name=exchangeName"`
	// ExchangeType is rabbitmq exchange type
	ExchangeType string `json:"exchangeType" protobuf:"bytes,3,opt,name=exchangeType"`
	// Routing key for bindings
	RoutingKey string `json:"routingKey" protobuf:"bytes,4,opt,name=routingKey"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,5,opt,name=connectionBackoff"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,6,opt,name=jsonBody"`
	// TLS configuration for the amqp client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,7,opt,name=tls"`
}

// KafkaEventSource refers to event-source for Kafka related events
type KafkaEventSource struct {
	// URL to kafka cluster
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Partition name
	Partition string `json:"partition" protobuf:"bytes,2,opt,name=partition"`
	// Topic name
	Topic string `json:"topic" protobuf:"bytes,3,opt,name=topic"`
	// Backoff holds parameters applied to connection.
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,4,opt,name=connectionBackoff"`
	// TLS configuration for the kafka client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,5,opt,name=tls"`
}

// MQTTEventSource refers to event-source for MQTT related events
type MQTTEventSource struct {
	// URL to connect to broker
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Topic name
	Topic string `json:"topic" protobuf:"bytes,2,opt,name=topic"`
	// ClientID is the id of the client
	ClientId string `json:"clientId" protobuf:"bytes,3,opt,name=clientId"`
	// ConnectionBackoff holds backoff applied to connection.
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,4,opt,name=connectionBackoff"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,5,opt,name=jsonBody"`
	// TLS configuration for the mqtt client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,6,opt,name=tls"`
}

// NATSEventSource refers to event-source for NATS related events
type NATSEventsSource struct {
	// URL to connect to NATS cluster
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Subject holds the name of the subject onto which messages are published
	Subject string `json:"subject" protobuf:"bytes,2,opt,name=subject"`
	// ConnectionBackoff holds backoff applied to connection.
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,3,opt,name=connectionBackoff"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,4,opt,name=jsonBody"`
	// TLS configuration for the nats client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,5,opt,name=tls"`
}

// SNSEventSource refers to event-source for AWS SNS related events
type SNSEventSource struct {
	// Webhook configuration for http server
	Webhook *Context `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`
	// TopicArn
	TopicArn string `json:"topicArn" protobuf:"bytes,2,opt,name=topicArn"`
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,3,opt,name=accessKey"`
	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,4,opt,name=secretKey"`
	// Namespace refers to Kubernetes namespace to read access related secret from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,5,opt,name=namespace"`
	// Region is AWS region
	Region string `json:"region" protobuf:"bytes,6,opt,name=region"`
	// RoleARN is the Amazon Resource Name (ARN) of the role to assume.
	// +optional
	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,7,opt,name=roleARN"`
}

// SQSEventSource refers to event-source for AWS SQS related events
type SQSEventSource struct {
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,1,opt,name=accessKey"`
	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,2,opt,name=secretKey"`
	// Region is AWS region
	Region string `json:"region" protobuf:"bytes,3,opt,name=region"`
	// Queue is AWS SQS queue to listen to for messages
	Queue string `json:"queue" protobuf:"bytes,4,opt,name=queue"`
	// WaitTimeSeconds is The duration (in seconds) for which the call waits for a message to arrive
	// in the queue before returning.
	WaitTimeSeconds int64 `json:"waitTimeSeconds" protobuf:"varint,5,opt,name=waitTimeSeconds"`
	// Namespace refers to Kubernetes namespace to read access related secret from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,6,opt,name=namespace"`
	// RoleARN is the Amazon Resource Name (ARN) of the role to assume.
	// +optional
	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,7,opt,name=roleARN"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,8,opt,name=jsonBody"`
	// QueueAccountId is the ID of the account that created the queue to monitor
	// +optional
	QueueAccountId string `json:"queueAccountId,omitempty" protobuf:"bytes,9,opt,name=queueAccountId"`
}

// PubSubEventSource refers to event-source for GCP PubSub related events.
type PubSubEventSource struct {
	// ProjectID is the unique identifier for your project on GCP
	ProjectID string `json:"projectID" protobuf:"bytes,1,opt,name=projectID"`
	// TopicProjectID identifies the project where the topic should exist or be created
	// (assumed to be the same as ProjectID by default)
	TopicProjectID string `json:"topicProjectID" protobuf:"bytes,2,opt,name=topicProjectID"`
	// Topic on which a subscription will be created
	Topic string `json:"topic" protobuf:"bytes,3,opt,name=topic"`
	// CredentialsFile is the file that contains credentials to authenticate for GCP
	CredentialsFile string `json:"credentialsFile" protobuf:"bytes,4,opt,name=credentialsFile"`
	// EnableWorkflowIdentity determines if your project authenticates to GCP with WorkflowIdentity or CredentialsFile.
	// If true, authentication is done with WorkflowIdentity. If false or omitted, authentication is done with CredentialsFile.
	// +optional
	EnableWorkflowIdentity bool `json:"enableWorkflowIdentity,omitempty" protobuf:"varint,5,opt,name=enableWorkflowIdentity"`
	// DeleteSubscriptionOnFinish determines whether to delete the GCP PubSub subscription once the event source is stopped.
	// +optional
	DeleteSubscriptionOnFinish bool `json:"deleteSubscriptionOnFinish,omitempty" protobuf:"varint,6,opt,name=deleteSubscriptionOnFinish"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,7,opt,name=jsonBody"`
}

// GithubEventSource refers to event-source for github related events
type GithubEventSource struct {
	// Id is the webhook's id
	Id int64 `json:"id" protobuf:"varint,1,opt,name=id"`
	// Webhook refers to the configuration required to run a http server
	Webhook *Context `json:"webhook,omitempty" protobuf:"bytes,2,opt,name=webhook"`
	// Owner refers to GitHub owner name i.e. argoproj
	Owner string `json:"owner" protobuf:"bytes,3,opt,name=owner"`
	// Repository refers to GitHub repo name i.e. argo-events
	Repository string `json:"repository" protobuf:"bytes,4,opt,name=repository"`
	// Events refer to Github events to subscribe to which the gateway will subscribe
	// +listType=string
	Events []string `json:"events" protobuf:"bytes,5,rep,name=events"`
	// APIToken refers to a K8s secret containing github api token
	APIToken *corev1.SecretKeySelector `json:"apiToken,omitempty" protobuf:"bytes,6,opt,name=apiToken"`
	// WebhookSecret refers to K8s secret containing GitHub webhook secret
	// https://developer.github.com/webhooks/securing/
	// +optional
	WebhookSecret *corev1.SecretKeySelector `json:"webhookSecret,omitempty" protobuf:"bytes,7,opt,name=webhookSecret"`
	// Insecure tls verification
	Insecure bool `json:"insecure,omitempty" protobuf:"varint,8,opt,name=insecure"`
	// Active refers to status of the webhook for event deliveries.
	// https://developer.github.com/webhooks/creating/#active
	// +optional
	Active bool `json:"active,omitempty" protobuf:"varint,9,opt,name=active"`
	// ContentType of the event delivery
	ContentType string `json:"contentType,omitempty" protobuf:"bytes,10,opt,name=contentType"`
	// GitHub base URL (for GitHub Enterprise)
	// +optional
	GithubBaseURL string `json:"githubBaseURL,omitempty" protobuf:"bytes,11,opt,name=githubBaseURL"`
	// GitHub upload URL (for GitHub Enterprise)
	// +optional
	GithubUploadURL string `json:"githubUploadURL,omitempty" protobuf:"bytes,12,opt,name=githubUploadURL"`
	// Namespace refers to Kubernetes namespace which is used to retrieve webhook secret and api token from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,13,opt,name=namespace"`
	// DeleteHookOnFinish determines whether to delete the GitHub hook for the repository once the event source is stopped.
	// +optional
	DeleteHookOnFinish bool `json:"deleteHookOnFinish,omitempty" protobuf:"varint,14,opt,name=deleteHookOnFinish"`
}

// GitlabEventSource refers to event-source related to Gitlab events
type GitlabEventSource struct {
	// Webhook holds configuration to run a http server
	Webhook *Context `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`
	// ProjectID is the id of project for which integration needs to setup
	ProjectID string `json:"projectId" protobuf:"bytes,2,opt,name=projectId"`
	// Events are gitlab event to listen to.
	// Refer https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794.
	Event string `json:"events" protobuf:"bytes,3,opt,name=event"`
	// AccessToken is reference to k8 secret which holds the gitlab api access information
	AccessToken *corev1.SecretKeySelector `json:"accessToken,omitempty" protobuf:"bytes,4,opt,name=accessToken"`
	// EnableSSLVerification to enable ssl verification
	// +optional
	EnableSSLVerification bool `json:"enableSSLVerification,omitempty" protobuf:"varint,5,opt,name=enableSSLVerification"`
	// GitlabBaseURL is the base URL for API requests to a custom endpoint
	GitlabBaseURL string `json:"gitlabBaseURL" protobuf:"bytes,6,opt,name=gitlabBaseURL"`
	// Namespace refers to Kubernetes namespace which is used to retrieve access token from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,7,opt,name=namespace"`
	// DeleteHookOnFinish determines whether to delete the GitLab hook for the project once the event source is stopped.
	// +optional
	DeleteHookOnFinish bool `json:"deleteHookOnFinish,omitempty" protobuf:"varint,8,opt,name=deleteHookOnFinish"`
	// AllowDuplicate allows the gateway to register the same webhook integrations for multiple event source configurations.
	// Defaults to false.
	// +optional.
	AllowDuplicate bool `json:"allowDuplicate,omitempty" protobuf:"varint,9,opt,name=allowDuplicate"`
}

// HDFSEventSource refers to event-source for HDFS related events
type HDFSEventSource struct {
	WatchPathConfig `json:",inline" protobuf:"bytes,1,opt,name=watchPathConfig"`
	// Type of file operations to watch
	Type string `json:"type" protobuf:"bytes,2,opt,name=type"`
	// CheckInterval is a string that describes an interval duration to check the directory state, e.g. 1s, 30m, 2h... (defaults to 1m)
	CheckInterval string `json:"checkInterval,omitempty" protobuf:"bytes,3,opt,name=checkInterval"`
	// Addresses is accessible addresses of HDFS name nodes
	// +listType=string
	Addresses []string `json:"addresses" protobuf:"bytes,4,rep,name=addresses"`
	// HDFSUser is the user to access HDFS file system.
	// It is ignored if either ccache or keytab is used.
	HDFSUser string `json:"hdfsUser,omitempty" protobuf:"bytes,5,opt,name=hdfsUser"`
	// KrbCCacheSecret is the secret selector for Kerberos ccache
	// Either ccache or keytab can be set to use Kerberos.
	KrbCCacheSecret *corev1.SecretKeySelector `json:"krbCCacheSecret,omitempty" protobuf:"bytes,6,opt,name=krbCCacheSecret"`
	// KrbKeytabSecret is the secret selector for Kerberos keytab
	// Either ccache or keytab can be set to use Kerberos.
	KrbKeytabSecret *corev1.SecretKeySelector `json:"krbKeytabSecret,omitempty" protobuf:"bytes,7,opt,name=krbKeytabSecret"`
	// KrbUsername is the Kerberos username used with Kerberos keytab
	// It must be set if keytab is used.
	KrbUsername string `json:"krbUsername,omitempty" protobuf:"bytes,8,opt,name=krbUsername"`
	// KrbRealm is the Kerberos realm used with Kerberos keytab
	// It must be set if keytab is used.
	KrbRealm string `json:"krbRealm,omitempty" protobuf:"bytes,9,opt,name=krbRealm"`
	// KrbConfig is the configmap selector for Kerberos config as string
	// It must be set if either ccache or keytab is used.
	KrbConfigConfigMap *corev1.ConfigMapKeySelector `json:"krbConfigConfigMap,omitempty" protobuf:"bytes,10,opt,name=krbConfigConfigMap"`
	// KrbServicePrincipalName is the principal name of Kerberos service
	// It must be set if either ccache or keytab is used.
	KrbServicePrincipalName string `json:"krbServicePrincipalName,omitempty" protobuf:"bytes,11,opt,name=krbServicePrincipalName"`
	// Namespace refers to Kubernetes namespace which is used to retrieve cache secret and ket tab secret from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,12,opt,name=namespace"`
}

// SlackEventSource refers to event-source for Slack related events
type SlackEventSource struct {
	// Slack App signing secret
	SigningSecret *corev1.SecretKeySelector `json:"signingSecret,omitempty" protobuf:"bytes,1,opt,name=signingSecret"`
	// Token for URL verification handshake
	Token *corev1.SecretKeySelector `json:"token,omitempty" protobuf:"bytes,2,opt,name=token"`
	// Webhook holds configuration for a REST endpoint
	Webhook *Context `json:"webhook,omitempty" protobuf:"bytes,3,opt,name=webhook"`
	// Namespace refers to Kubernetes namespace which is used to retrieve token and signing secret from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
}

// StorageGridEventSource refers to event-source for StorageGrid related events
type StorageGridEventSource struct {
	// Webhook holds configuration for a REST endpoint
	Webhook *Context `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`
	// Events are s3 bucket notification events.
	// For more information on s3 notifications, follow https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations
	// Note that storage grid notifications do not contain `s3:`
	// +listType=string
	Events []string `json:"events,omitempty" protobuf:"bytes,2,rep,name=events"`
	// Filter on object key which caused the notification.
	Filter *StorageGridFilter `json:"filter,omitempty" protobuf:"bytes,3,opt,name=filter"`
	// TopicArn
	TopicArn string `json:"topicArn" protobuf:"bytes,4,name=topicArn"`
	// Name of the bucket to register notifications for.
	Bucket string `json:"bucket" protobuf:"bytes,5,name=bucket"`
	// S3 region.
	// Defaults to us-east-1
	// +optional
	Region string `json:"region,omitempty" protobuf:"bytes,6,opt,name=region"`
	// Auth token for storagegrid api
	AuthToken *corev1.SecretKeySelector `json:"authToken" protobuf:"bytes,7,name=authToken"`
	// ApiURL is the url of the storagegrid api.
	ApiURL string `json:"apiURL" protobuf:"bytes,8,name=apiURL"`
}

// Filter represents filters to apply to bucket notifications for specifying constraints on objects
// +k8s:openapi-gen=true
type StorageGridFilter struct {
	Prefix string `json:"prefix" protobuf:"bytes,1,opt,name=prefix"`
	Suffix string `json:"suffix" protobuf:"bytes,2,opt,name=suffix"`
}

// AzureEventsHubEventSource describes the event source for azure events hub
// More info at https://docs.microsoft.com/en-us/azure/event-hubs/
type AzureEventsHubEventSource struct {
	// FQDN of the EventHubs namespace you created
	// More info at https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string
	FQDN string `json:"fqdn" protobuf:"bytes,1,opt,name=fqdn"`
	// SharedAccessKeyName is the name you chose for your application's SAS keys
	SharedAccessKeyName *corev1.SecretKeySelector `json:"sharedAccessKeyName,omitempty" protobuf:"bytes,2,opt,name=sharedAccessKeyName"`
	// SharedAccessKey is the the generated value of the key
	SharedAccessKey *corev1.SecretKeySelector `json:"sharedAccessKey,omitempty" protobuf:"bytes,3,opt,name=sharedAccessKey"`
	// Event Hub path/name
	HubName string `json:"hubName" protobuf:"bytes,4,opt,name=hubName"`
	// Namespace refers to Kubernetes namespace which is used to retrieve the shared access key and name from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,5,opt,name=namespace"`
}

// StripeEventSource describes the event source for stripe webhook notifications
// More info at https://stripe.com/docs/webhooks
type StripeEventSource struct {
	// Webhook holds configuration for a REST endpoint
	Webhook *Context `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`
	// CreateWebhook if specified creates a new webhook programmatically.
	// +optional
	CreateWebhook bool `json:"createWebhook,omitempty" protobuf:"varint,2,opt,name=createWebhook"`
	// APIKey refers to K8s secret that holds Stripe API key. Used only if CreateWebhook is enabled.
	// +optional
	APIKey *corev1.SecretKeySelector `json:"apiKey,omitempty" protobuf:"bytes,3,opt,name=apiKey"`
	// Namespace to retrieve the APIKey secret from. Must be specified in order to read API key from APIKey K8s secret.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
	// EventFilter describes the type of events to listen to. If not specified, all types of events will be processed.
	// More info at https://stripe.com/docs/api/events/list
	// +optional
	// +listType=string
	EventFilter []string `json:"eventFilter,omitempty" protobuf:"bytes,5,rep,name=eventFilter"`
}

// EmitterEventSource describes the event source for emitter
// More info at https://emitter.io/develop/getting-started/
type EmitterEventSource struct {
	// Broker URI to connect to.
	Broker string `json:"broker" protobuf:"bytes,1,opt,name=broker"`
	// ChannelKey refers to the channel key
	ChannelKey string `json:"channelKey" protobuf:"bytes,2,opt,name=channelKey"`
	// ChannelName refers to the channel name
	ChannelName string `json:"channelName" protobuf:"bytes,3,opt,name=channelName"`
	// Namespace to use to retrieve the channel key and optional username/password
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
	// Username to use to connect to broker
	// +optional
	Username *corev1.SecretKeySelector `json:"username,omitempty" protobuf:"bytes,5,opt,name=username"`
	// Password to use to connect to broker
	// +optional
	Password *corev1.SecretKeySelector `json:"password,omitempty" protobuf:"bytes,6,opt,name=password"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,7,opt,name=connectionBackoff"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,8,opt,name=jsonBody"`
	// TLS configuration for the emitter client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,9,opt,name=tls"`
}

// RedisEventSource describes an event source for the Redis PubSub.
// More info at https://godoc.org/github.com/go-redis/redis#example-PubSub
type RedisEventSource struct {
	// HostAddress refers to the address of the Redis host/server
	HostAddress string `json:"hostAddress" protobuf:"bytes,1,opt,name=hostAddress"`
	// Password required for authentication if any.
	// +optional
	Password *corev1.SecretKeySelector `json:"password,omitempty" protobuf:"bytes,2,opt,name=password"`
	// Namespace to use to retrieve the password from. It should only be specified if password is declared
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	// DB to use. If not specified, default DB 0 will be used.
	// +optional
	DB int32 `json:"db,omitempty" protobuf:"varint,4,opt,name=db"`
	// Channels to subscribe to listen events.
	// +listType=string
	Channels []string `json:"channels" protobuf:"bytes,5,rep,name=channels"`
	// TLS configuration for the redis client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,6,opt,name=tls"`
}

// NSQEventSource describes the event source for NSQ PubSub
// More info at https://godoc.org/github.com/nsqio/go-nsq
type NSQEventSource struct {
	// HostAddress is the address of the host for NSQ lookup
	HostAddress string `json:"hostAddress" protobuf:"bytes,1,opt,name=hostAddress"`
	// Topic to subscribe to.
	Topic string `json:"topic" protobuf:"bytes,2,opt,name=topic"`
	// Channel used for subscription
	Channel string `json:"channel" protobuf:"bytes,3,opt,name=channel"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,4,opt,name=connectionBackoff"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,5,opt,name=jsonBody"`
	// TLS configuration for the nsq client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,6,opt,name=tls"`
}

// GenericEventSource refers to a generic event source. It can be used to implement a custom event source.
type GenericEventSource struct {
	// Value of the event source
	Value string `json:"value" protobuf:"bytes,1,opt,name=value"`
}

// TLSConfig refers to TLS configuration for a client.
type TLSConfig struct {
	// CACertPath refers the file path that contains the CA cert.
	CACertPath string `json:"caCertPath" protobuf:"bytes,1,opt,name=caCertPath"`
	// ClientCertPath refers the file path that contains client cert.
	ClientCertPath string `json:"clientCertPath" protobuf:"bytes,2,opt,name=clientCertPath"`
	// ClientKeyPath refers the file path that contains client key.
	ClientKeyPath string `json:"clientKeyPath" protobuf:"bytes,3,opt,name=clientKeyPath"`
}

// EventSourceStatus holds the status of the event-source resource
type EventSourceStatus struct {
	CreatedAt metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,1,opt,name=createdAt"`
}
