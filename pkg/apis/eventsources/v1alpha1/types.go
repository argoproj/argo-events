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

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways/server/common/fsevent"
	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventSource is the definition of a eventsource resource
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type EventSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Status            EventSourceStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	Spec              *EventSourceSpec  `json:"spec" protobuf:"bytes,3,opt,name=spec"`
}

// EventSourceList is the list of eventsource resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EventSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	// +listType=eventsource
	Items []EventSource `json:"items" protobuf:"bytes,2,opt,name=items"`
}

// EventSourceSpec refers to specification of event-source resource
type EventSourceSpec struct {
	// Minio event sources
	Minio map[string]apicommon.S3Artifact `json:"minio,omitempty" protobuf:"bytes,1,opt,name=minio"`
	// Calendar event sources
	Calendar map[string]CalendarEventSource `json:"calendar,omitempty" protobuf:"bytes,2,opt,name=calendar"`
	// File event sources
	File map[string]FileEventSource `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`
	// Resource event sources
	Resource map[string]ResourceEventSource `json:"resource,omitempty" protobuf:"bytes,4,opt,name=resource"`
	// Webhook event sources
	Webhook map[string]webhook.Context `json:"webhook,omitempty" protobuf:"bytes,5,opt,name=webhook"`
	// AMQP event sources
	AMQP map[string]AMQPEventSource `json:"amqp,omitempty" protobuf:"bytes,6,opt,name=amqp"`
	// Kafka event sources
	Kafka map[string]KafkaEventSource `json:"kafka,omitempty" protobuf:"bytes,7,opt,name=kafka"`
	// MQTT event sources
	MQTT map[string]MQTTEventSource `json:"mqtt,omitempty" protobuf:"bytes,8,opt,name=mqtt"`
	// NATS event sources
	NATS map[string]NATSEventsSource `json:"nats,omitempty" protobuf:"bytes,9,opt,name=nats"`
	// SNS event sources
	SNS map[string]SNSEventSource `json:"sns,omitempty" protobuf:"bytes,10,opt,name=sns"`
	// SQS event sources
	SQS map[string]SQSEventSource `json:"sqs,omitempty" protobuf:"bytes,11,opt,name=sqs"`
	// PubSub eevnt sources
	PubSub map[string]PubSubEventSource `json:"pubSub,omitempty" protobuf:"bytes,12,opt,name=pubSub"`
	// Github event sources
	Github map[string]GithubEventSource `json:"github,omitempty" protobuf:"bytes,13,opt,name=github"`
	// Gitlab event sources
	Gitlab map[string]GitlabEventSource `json:"gitlab,omitempty" protobuf:"bytes,14,opt,name=gitlab"`
	// HDFS event sources
	HDFS map[string]HDFSEventSource `json:"hdfs,omitempty" protobuf:"bytes,15,opt,name=hdfs"`
	// Slack event sources
	Slack map[string]SlackEventSource `json:"slack,omitempty" protobuf:"bytes,16,opt,name=slack"`
	// StorageGrid event sources
	StorageGrid map[string]StorageGridEventSource `json:"storageGrid,omitempty" protobuf:"bytes,17,opt,name=storageGrid"`
	// Type of the event source
	Type apicommon.EventSourceType `json:"type" protobuf:"bytes,19,name=type"`
}

// CalendarEventSource describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type CalendarEventSource struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule" protobuf:"bytes,1,name=schedule"`
	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval" protobuf:"bytes,2,name=interval"`
	// ExclusionDates defines the list of DATE-TIME exceptions for recurring events.
	// +listType=string
	ExclusionDates []string `json:"exclusionDates,omitempty" protobuf:"bytes,3,opt,name=exclusionDates"`
	// Timezone in which to run the schedule
	// +optional
	Timezone string `json:"timezone,omitempty" protobuf:"bytes,4,opt,name=timezone"`
	// UserPayload will be sent to sensor as extra data once the event is triggered
	// +optional
	UserPayload *json.RawMessage `json:"userPayload,omitempty" protobuf:"bytes,5,opt,name=userPayload"`
}

// FileEventSource describes an event-source for file related events.
type FileEventSource struct {
	// Type of file operations to watch
	// Refer https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go for more information
	EventType string `json:"eventType" protobuf:"bytes,1,name=eventType"`
	// WatchPathConfig contains configuration about the file path to watch
	WatchPathConfig fsevent.WatchPathConfig `json:"watchPathConfig" protobuf:"bytes,2,name=watchPathConfig"`
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
	Namespace string `json:"namespace" protobuf:"bytes,1,name=namespace"`
	// Filter is applied on the metadata of the resource
	// +optional
	Filter *ResourceFilter `json:"filter,omitempty" protobuf:"bytes,2,opt,name=filter"`
	// Group of the resource
	metav1.GroupVersionResource `json:",inline"`
	// Type is the event type.
	// If not provided, the gateway will watch all events for a resource.
	// +optional
	EventType ResourceEventType `json:"eventType,omitempty" protobuf:"bytes,3,opt,name=eventType"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource event objects
type ResourceFilter struct {
	// +optional
	Prefix string `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,2,opt,name=labels"`
	// +optional
	Fields map[string]string `json:"fields,omitempty" protobuf:"bytes,3,opt,name=fields"`
	// +optional
	CreatedBy metav1.Time `json:"createdBy,omitempty" protobuf:"bytes,4,opt,name=createdBy"`
}

// AMQPEventSource refers to an event-source for AMQP stream events
type AMQPEventSource struct {
	// URL for rabbitmq service
	URL string `json:"url" protobuf:"bytes,1,name=url"`
	// ExchangeName is the exchange name
	// For more information, visit https://www.rabbitmq.com/tutorials/amqp-concepts.html
	ExchangeName string `json:"exchangeName" protobuf:"bytes,2,name=exchangeName"`
	// ExchangeType is rabbitmq exchange type
	ExchangeType string `json:"exchangeType" protobuf:"bytes,3,name=exchangeType"`
	// Routing key for bindings
	RoutingKey string `json:"routingKey" protobuf:"bytes,4,name=routingKey"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *common.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,5,opt,name=connectionBackoff"`
}

// KafkaEventSource refers to event-source for Kafka related events
type KafkaEventSource struct {
	// URL to kafka cluster
	URL string `json:"url" protobuf:"bytes,1,name=url"`
	// Partition name
	Partition string `json:"partition" protobuf:"bytes,2,name=partition"`
	// Topic name
	Topic string `json:"topic" protobuf:"bytes,3,name=topic"`
	// Backoff holds parameters applied to connection.
	ConnectionBackoff *common.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,4,opt,name=connectionBackoff"`
}

// MQTTEventSource refers to event-source for MQTT related events
type MQTTEventSource struct {
	// URL to connect to broker
	URL string `json:"url" protobuf:"bytes,1,name=url"`
	// Topic name
	Topic string `json:"topic" protobuf:"bytes,2,name=topic"`
	// ClientID is the id of the client
	ClientId string `json:"clientId" protobuf:"bytes,3,name=clientId"`
	// ConnectionBackoff holds backoff applied to connection.
	ConnectionBackoff *common.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,4,opt,name=connectionBackoff"`
}

// NATSEventSource refers to event-source for NATS related events
type NATSEventsSource struct {
	// URL to connect to NATS cluster
	URL string `json:"url" protobuf:"bytes,1,name=url"`
	// Subject holds the name of the subject onto which messages are published
	Subject string `json:"subject" protobuf:"bytes,2,name=2"`
	// ConnectionBackoff holds backoff applied to connection.
	ConnectionBackoff *common.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,3,opt,name=connectionBackoff"`
}

// SNSEventSource refers to event-source for AWS SNS related events
type SNSEventSource struct {
	// Webhook configuration for http server
	Webhook *webhook.Context `json:"webhook" protobuf:"bytes,1,name=webhook"`
	// TopicArn
	TopicArn string `json:"topicArn" protobuf:"bytes,2,name=topicArn"`
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,3,opt,name=accessKey"`
	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,4,opt,name=secretKey"`
	// Namespace refers to Kubernetes namespace to read access related secret from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,5,opt,name=namespace"`
	// Region is AWS region
	Region string `json:"region" protobuf:"bytes,6,name=region"`
}

// SQSEventSource refers to event-source for AWS SQS related events
type SQSEventSource struct {
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,1,opt,name=accessKey"`
	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,2,opt,name=accessKey"`
	// Region is AWS region
	Region string `json:"region" protobuf:"bytes,3,name=region"`
	// Queue is AWS SQS queue to listen to for messages
	Queue string `json:"queue" protobuf:"bytes,4,name=queue"`
	// WaitTimeSeconds is The duration (in seconds) for which the call waits for a message to arrive
	// in the queue before returning.
	WaitTimeSeconds int64 `json:"waitTimeSeconds" protobuf:"bytes,5,name=waitTimeSeconds"`
	// Namespace refers to Kubernetes namespace to read access related secret from.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,6,opt,name=namespace"`
}

// PubSubEventSource refers to event-source for GCP PubSub related events.
type PubSubEventSource struct {
	// ProjectID is the unique identifier for your project on GCP
	ProjectID string `json:"projectID" protobuf:"bytes,1,name=projectID"`
	// TopicProjectID identifies the project where the topic should exist or be created
	// (assumed to be the same as ProjectID by default)
	TopicProjectID string `json:"topicProjectID" protobuf:"bytes,2,name=topicProjectID"`
	// Topic on which a subscription will be created
	Topic string `json:"topic" protobuf:"bytes,3,name=topic"`
	// CredentialsFile is the file that contains credentials to authenticate for GCP
	CredentialsFile string `json:"credentialsFile" protobuf:"bytes,4,name=credentialsFile"`
	// DeleteSubscriptionOnFinish determines whether to delete the GCP PubSub subscription once the event source is stopped.
	// +optional
	DeleteSubscriptionOnFinish bool `json:"deleteSubscriptionOnFinish,omitempty" protobuf:"bytes,1,opt,name=deleteSubscriptionOnFinish"`
}

// GithubEventSource refers to event-source for github related events
type GithubEventSource struct {
	// Id is the webhook's id
	Id int64 `json:"id" protobuf:"bytes,1,name=id"`
	// Webhook refers to the configuration required to run a http server
	Webhook *webhook.Context `json:"webhook" protobuf:"bytes,2,name=webhook"`
	// Owner refers to GitHub owner name i.e. argoproj
	Owner string `json:"owner" protobuf:"bytes,3,name=owner"`
	// Repository refers to GitHub repo name i.e. argo-events
	Repository string `json:"repository" protobuf:"bytes,4,name=repository"`
	// Events refer to Github events to subscribe to which the gateway will subscribe
	// +listType=string
	Events []string `json:"events" protobuf:"bytes,5,rep,name=events"`
	// APIToken refers to a K8s secret containing github api token
	APIToken *corev1.SecretKeySelector `json:"apiToken"`
	// WebhookSecret refers to K8s secret containing GitHub webhook secret
	// https://developer.github.com/webhooks/securing/
	// +optional
	WebhookSecret *corev1.SecretKeySelector `json:"webhookSecret,omitempty" protobuf:"bytes,7,opt,name=webhookSecret"`
	// Insecure tls verification
	Insecure bool `json:"insecure,omitempty" protobuf:"bytes,8,opt,name=insecure"`
	// Active refers to status of the webhook for event deliveries.
	// https://developer.github.com/webhooks/creating/#active
	// +optional
	Active bool `json:"active,omitempty" protobuf:"bytes,9,opt,name=active"`
	// ContentType of the event delivery
	ContentType string `json:"contentType,omitempty" protobuf:"bytes,10,opt,name=contentType"`
	// GitHub base URL (for GitHub Enterprise)
	// +optional
	GithubBaseURL string `json:"githubBaseURL,omitempty" protobuf:"bytes,11,opt,name=githubBaseURL"`
	// GitHub upload URL (for GitHub Enterprise)
	// +optional
	GithubUploadURL string `json:"githubUploadURL,omitempty" protobuf:"bytes,12,opt,name=githubUploadURL"`
	// Namespace refers to Kubernetes namespace which is used to retrieve webhook secret and api token from.
	Namespace string `json:"namespace" protobuf:"bytes,13,name=namespace"`
	// DeleteHookOnFinish determines whether to delete the GitHub hook for the repository once the event source is stopped.
	// +optional
	DeleteHookOnFinish bool `json:"deleteHookOnFinish,omitempty" protobuf:"bytes,14,opt,name=deleteHookOnFinish"`
}

// GitlabEventSource refers to event-source related to Gitlab events
type GitlabEventSource struct {
	// Webhook holds configuration to run a http server
	Webhook *webhook.Context `json:"webhook" protobuf:"bytes,1,name=webhook"`
	// ProjectId is the id of project for which integration needs to setup
	ProjectId string `json:"projectId" protobuf:"bytes,2,name=projectId"`
	// Event is a gitlab event to listen to.
	// Refer https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794.
	Event string `json:"event" protobuf:"bytes,3,name=event"`
	// AccessToken is reference to k8 secret which holds the gitlab api access information
	AccessToken *corev1.SecretKeySelector `json:"accessToken" protobuf:"bytes,4,name=accessToken"`
	// EnableSSLVerification to enable ssl verification
	// +optional
	EnableSSLVerification bool `json:"enableSSLVerification,omitempty" protobuf:"bytes,5,opt,name=enableSSLVerification"`
	// GitlabBaseURL is the base URL for API requests to a custom endpoint
	GitlabBaseURL string `json:"gitlabBaseURL" protobuf:"bytes,6,name=gitlabBaseURL"`
	// Namespace refers to Kubernetes namespace which is used to retrieve access token from.
	Namespace string `json:"namespace" protobuf:"bytes,7,name=namespace"`
	// DeleteHookOnFinish determines whether to delete the GitLab hook for the project once the event source is stopped.
	// +optional
	DeleteHookOnFinish bool `json:"deleteHookOnFinish,omitempty" protobuf:"bytes,8,opt,name=deleteHookOnFinish"`
}

// HDFSEventSource refers to event-source for HDFS related events
type HDFSEventSource struct {
	fsevent.WatchPathConfig `json:",inline"`
	// Type of file operations to watch
	Type string `json:"type"`
	// CheckInterval is a string that describes an interval duration to check the directory state, e.g. 1s, 30m, 2h... (defaults to 1m)
	CheckInterval string `json:"checkInterval,omitempty"`
	// Addresses is accessible addresses of HDFS name nodes
	// +listType=string
	Addresses []string `json:"addresses"`
	// HDFSUser is the user to access HDFS file system.
	// It is ignored if either ccache or keytab is used.
	HDFSUser string `json:"hdfsUser,omitempty"`
	// KrbCCacheSecret is the secret selector for Kerberos ccache
	// Either ccache or keytab can be set to use Kerberos.
	KrbCCacheSecret *corev1.SecretKeySelector `json:"krbCCacheSecret,omitempty"`
	// KrbKeytabSecret is the secret selector for Kerberos keytab
	// Either ccache or keytab can be set to use Kerberos.
	KrbKeytabSecret *corev1.SecretKeySelector `json:"krbKeytabSecret,omitempty"`
	// KrbUsername is the Kerberos username used with Kerberos keytab
	// It must be set if keytab is used.
	KrbUsername string `json:"krbUsername,omitempty"`
	// KrbRealm is the Kerberos realm used with Kerberos keytab
	// It must be set if keytab is used.
	KrbRealm string `json:"krbRealm,omitempty"`
	// KrbConfig is the configmap selector for Kerberos config as string
	// It must be set if either ccache or keytab is used.
	KrbConfigConfigMap *corev1.ConfigMapKeySelector `json:"krbConfigConfigMap,omitempty"`
	// KrbServicePrincipalName is the principal name of Kerberos service
	// It must be set if either ccache or keytab is used.
	KrbServicePrincipalName string `json:"krbServicePrincipalName,omitempty"`
	// Namespace refers to Kubernetes namespace which is used to retrieve cache secret and ket tab secret from.
	Namespace string `json:"namespace" protobuf:"bytes,1,name=namespace"`
}

// SlackEventSource refers to event-source for Slack related events
type SlackEventSource struct {
	// Slack App signing secret
	SigningSecret *corev1.SecretKeySelector `json:"signingSecret,omitempty" protobuf:"bytes,1,opt,name=signingSecret"`
	// Token for URL verification handshake
	Token *corev1.SecretKeySelector `json:"token,omitempty" protobuf:"bytes,2,name=token"`
	// Webhook holds configuration for a REST endpoint
	Webhook *webhook.Context `json:"webhook" protobuf:"bytes,3,name=webhook"`
	// Namespace refers to Kubernetes namespace which is used to retrieve token and signing secret from.
	Namespace string `json:"namespace" protobuf:"bytes,4,name=namespace"`
}

// StorageGridEventSource refers to event-source for StorageGrid related events
type StorageGridEventSource struct {
	// Webhook holds configuration for a REST endpoint
	Webhook *webhook.Context `json:"webhook" protobuf:"bytes,1,name=webhook"`
	// Events are s3 bucket notification events.
	// For more information on s3 notifications, follow https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations
	// Note that storage grid notifications do not contain `s3:`
	// +listType=string
	Events []string `json:"events,omitempty" protobuf:"bytes,2,opt,name=events"`
	// Filter on object key which caused the notification.
	Filter *StorageGridFilter `json:"filter,omitempty" protobuf:"bytes,3,opt,name=filter"`
}

// Filter represents filters to apply to bucket notifications for specifying constraints on objects
// +k8s:openapi-gen=true
type StorageGridFilter struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

// EventSourceStatus holds the status of the event-source resource
type EventSourceStatus struct {
	CreatedAt metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,1,opt,name=createdAt"`
}
