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
	// AzureEventsHub event sources
	AzureEventsHub map[string]AzureEventsHubEventSource `json:"azureEventsHub,omitempty" protobuf:"bytes,18,opt,name=azureEventsHub"`
	// Stripe event sources
	Stripe map[string]StripeEventSource `json:"stripe,omitempty" protobuf:"bytes,19,opt,name=stripe"`
	// Emitter event source
	Emitter map[string]EmitterEventSource `json:"emitter,omitempty" protobuf:"bytes,20,opt,name=emitter"`
	// Redis event source
	Redis map[string]RedisEventSource `json:"redis,omitempty" protobuf:"bytes,21,opt,name=redis"`
	// NSQ event source
	NSQ map[string]NSQEventSource `json:"nsq,omitempty" protobuf:"bytes,22,opt,name=nsq"`
	// Generic event source
	Generic map[string]GenericEventSource `json:"generic,omitempty" protobuf:"bytes,23,opt,name=generic"`
	// Type of the event source
	Type apicommon.EventSourceType `json:"type" protobuf:"bytes,24,name=type"`
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

// AzureEventsHubEventSource describes the event source for azure events hub
// More info at https://docs.microsoft.com/en-us/azure/event-hubs/
type AzureEventsHubEventSource struct {
	// FQDN of the EventHubs namespace you created
	// More info at https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string
	FQDN string `json:"fqdn" protobuf:"bytes,1,name=fqdn"`
	// SharedAccessKeyName is the name you chose for your application's SAS keys
	SharedAccessKeyName *corev1.SecretKeySelector `json:"sharedAccessKeyName" protobuf:"bytes,2,name=sharedAccessKeyName"`
	// SharedAccessKey is the the generated value of the key
	SharedAccessKey *corev1.SecretKeySelector `json:"sharedAccessKey" protobuf:"bytes,3,name=sharedAccessKey"`
	// Event Hub path/name
	HubName string `json:"hubName" protobuf:"bytes,4,name=hubName"`
	// Namespace refers to Kubernetes namespace which is used to retrieve the shared access key and name from.
	Namespace string `json:"namespace" protobuf:"bytes,5,name=namespace"`
}

// StripeEventSource describes the event source for stripe webhook notifications
// More info at https://stripe.com/docs/webhooks
type StripeEventSource struct {
	// Webhook holds configuration for a REST endpoint
	Webhook *webhook.Context `json:"webhook" protobuf:"bytes,1,name=webhook"`
	// CreateWebhook if specified creates a new webhook programmatically.
	// +optional
	CreateWebhook bool `json:"createWebhook,omitempty" protobuf:"bytes,2,opt,name=createWebhook"`
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
	Broker string `json:"broker" protobuf:"bytes,1,name=broker"`
	// ChannelKey refers to the channel key
	ChannelKey *corev1.SecretKeySelector `json:"channelKey" protobuf:"bytes,2,name=channelKey"`
	// ChannelName refers to the channel name
	ChannelName string `json:"channelName" protobuf:"bytes,3,name=channelName"`
	// Namespace to use to retrieve the channel key and optional username/password
	Namespace string `json:"namespace" protobuf:"bytes,4,name=namespace"`
	// Username to use to connect to broker
	// +optional
	Username *corev1.SecretKeySelector `json:"username,omitempty" protobuf:"bytes,5,opt,name=username"`
	// Password to use to connect to broker
	// +optional
	Password *corev1.SecretKeySelector `json:"password,omitempty" protobuf:"bytes,6,opt,name=password"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *common.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,7,opt,name=connectionBackoff"`
}

// RedisEventSource describes an event source for the Redis PubSub.
// More info at https://godoc.org/github.com/go-redis/redis#example-PubSub
type RedisEventSource struct {
	// HostAddress refers to the address of the Redis host/server
	HostAddress string `json:"hostAddress" protobuf:"bytes,1,name=hostAddress"`
	// Password required for authentication if any.
	// +optional
	Password *corev1.SecretKeySelector `json:"password,omitempty" protobuf:"bytes,2,opt,name=password"`
	// Namespace to use to retrieve the password from. It should only be specified if password is declared
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	// DB to use. If not specified, default DB 0 will be used.
	// +optional
	DB int `json:"db,omitempty" protobuf:"bytes,4,opt,name=db"`
	// Channels to subscribe to listen events.
	// +listType=string
	Channels []string `json:"channels" protobuf:"bytes,5,name=channels"`
}

// NSQEventSource describes the event source for NSQ PubSub
// More info at https://godoc.org/github.com/nsqio/go-nsq
type NSQEventSource struct {
	// HostAddress is the address of the host for NSQ lookup
	HostAddress string `json:"hostAddress" protobuf:"bytes,1,name=hostAddress"`
	// Topic to subscribe to.
	Topic string `json:"topic" protobuf:"bytes,2,name=topic"`
	// Channel used for subscription
	Channel string `json:"channel" protobuf:"bytes,3,name=channel"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *common.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,4,opt,name=connectionBackoff"`
}

// GenericEventSource refers to a generic event source. It can be used to implement a custom event source.
type GenericEventSource struct {
	// Value of the event source
	Value string `json:"value" protobuf:"bytes,1,name=value"`
}

// EventSourceStatus holds the status of the event-source resource
type EventSourceStatus struct {
	CreatedAt metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,1,opt,name=createdAt"`
}
