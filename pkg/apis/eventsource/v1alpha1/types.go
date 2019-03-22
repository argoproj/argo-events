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
	"github.com/argoproj/argo-events/gateways/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
)

// SourceType is the type of event source
type SourceType string

// Possible values for event source
// These must match the field names from `Source`
const (
	SourceTypeWebhook     SourceType = "webbook"
	SourceTypeArtifact    SourceType = "artifact"
	SourceTypeCalendar    SourceType = "calendar"
	SourceTypeFile        SourceType = "file"
	SourceTypeResource    SourceType = "resource"
	SourceTypeAMQP        SourceType = "amqp"
	SourceTypeKafka       SourceType = "kafka"
	SourceTypeMQTT        SourceType = "mqtt"
	SourceTypeNats        SourceType = "nats"
	SourceTypeSNS         SourceType = "sns"
	SourceTypeSQS         SourceType = "sqs"
	SourceTypePubSub      SourceType = "pubSub"
	SourceTypeGithub      SourceType = "github"
	SourceTypeGitlab      SourceType = "gitlab"
	SourceTypeHDFS        SourceType = "hdfs"
	SourceTypeSlack       SourceType = "slack"
	SourceTypeStorageGrid SourceType = "storageGrid"
)

// EventSource is the definition of an event source
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type EventSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Status            metav1.Status   `json:"status" protobuf:"bytes,2,opt,name=status"`
	Spec              EventSourceSpec `json:"spec" protobuf:"bytes,3,opt,name=spec"`
}

// EventSourceSpec represents event source specifications
type EventSourceSpec struct {
	Source Source     `json:"source" protobuf:"bytes,1,opt,name=source"`
	Type   SourceType `json:"type" protobuf:"bytes,2,opt,name=type"`
}

type Source struct {
	Webhook map[string]WebhookEventSource `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`

	Artifact map[string]S3Artifact `json:"artifact,omitempty" protobuf:"bytes,2,opt,name=artifact"`

	Calendar map[string]CalendarEventSource `json:"calendar,omitempty" protobuf:"bytes,3,opt,name=calendar"`

	File map[string]FileEventSource `json:"file,omitempty" protobuf:"bytes,4,opt,name=file"`

	Resource map[string]ResourceEventSource `json:"resource,omitempty" protobuf:"bytes,5,opt,name=resource"`

	AMQP map[string]AMQPEventSource `json:"amqp,omitempty" protobuf:"bytes,6,opt,name=amqp"`

	Kakfa map[string]KafkaEventSource `json:"kakfa,omitempty" protobuf:"bytes,7,opt,name=kafka"`

	MQTT map[string]MQTTEventSource `json:"mqtt,omitempty" protobuf:"bytes,8,opt,name=mqtt"`

	Nats map[string]NATSEventSource `json:"nats,omitempty" protobuf:"bytes,9,opt,name=nats"`

	SNS map[string]SNSEventSource `json:"sns,omitempty" protobuf:"bytes,10,opt,name=sns"`

	SQS map[string]SQSEventSource `json:"sqs,omitempty" protobuf:"bytes,11,opt,name=sqs"`

	PubSub map[string]PubSubEventSource `json:"pubSub,omitempty" protobuf:"bytes,12,opt,name=pubSub"`

	Github map[string]GithubEventSource `json:"github,omitempty" protobuf:"bytes,13,opt,name=github"`

	Gitlab map[string]GitlabEventSource `json:"gitlab,omitempty" protobuf:"bytes,14,opt,name=gitlab"`

	HDFS map[string]HDFSEventSource `json:"hdfs,omitempty" protobuf:"bytes,15,opt,name=hdfs"`

	Slack map[string]SlackEventSource `json:"slack,omitempty" protobuf:"bytes,16,opt,name=slack"`

	StorageGrid map[string]StorageGridEventSource `json:"storageGrid,omitempty" protobuf:"bytes,17,opt,name=storageGrid"`
}

// WebhookEventSource contains general purpose REST API configuration
type WebhookEventSource struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,name=endpoint"`
	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,2,name=method"`
	// Port on which HTTP server is listening for incoming events.
	Port string `json:"port" protobuf:"bytes,3,name=port"`
	// URL is the url of the server.
	URL string `json:"url" protobuf:"bytes,4,name=url"`
	// ServerCertPath refers the file that contains the cert.
	ServerCertPath string `json:"serverCertPath,omitempty" protobuf:"bytes,4,opt,name=serverCertPath"`
	// ServerKeyPath refers the file that contains private key
	ServerKeyPath string `json:"serverKeyPath,omitempty" protobuf:"bytes,5,opt,name=serverKeyPath"`
}

// S3ArtifactEventSource contains information about an artifact in S3
type S3Artifact struct {
	Endpoint  string                    `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`
	Bucket    *S3Bucket                 `json:"bucket" protobuf:"bytes,2,opt,name=bucket"`
	Region    string                    `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
	Insecure  bool                      `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`
	AccessKey *corev1.SecretKeySelector `json:"accessKey" protobuf:"bytes,5,opt,name=accessKey"`
	SecretKey *corev1.SecretKeySelector `json:"secretKey" protobuf:"bytes,6,opt,name=secretKey"`
	Events    []string                  `json:"events,omitempty" protobuf:"bytes,7,opt,name=events"`
	Filter    *S3Filter                 `json:"filter,omitempty" protobuf:"bytes,8,opt,name=filter"`
}

// S3Bucket contains information to describe an S3 Bucket
type S3Bucket struct {
	Key  string `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
}

// S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects
type S3Filter struct {
	Prefix string `json:"prefix" protobuf:"bytes,1,opt,name=prefix"`
	Suffix string `json:"suffix" protobuf:"bytes,2,opt,name=suffix"`
}

// CalendarEventSource describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
// +k8s:openapi-gen=true
type CalendarEventSource struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule"`

	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval"`

	// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
	// RDATE defines the list of DATE-TIME values for recurring events.
	// EXDATE defines the list of DATE-TIME exceptions for recurring events.
	// the combination of these rules and dates combine to form a set of date times.
	// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
	Recurrence []string `json:"recurrence,omitempty"`

	// Timezone in which to run the schedule
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// UserPayload will be sent to sensor as extra data once the event is triggered
	// +optional
	UserPayload string `json:"userPayload,omitempty"`
}

// FileEventSource is the event source for file gateway
type FileEventSource struct {
	WatchPathConfig `json:",inline"`

	// Type of file operations to watch
	// Refer https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go for more information
	Type string `json:"type"`
}

type WatchPathConfig struct {
	// Directory to watch for events
	Directory string `json:"directory"`
	// Path is relative path of object to watch with respect to the directory
	Path string `json:"path,omitempty"`
	// PathRegexp is regexp of relative path of object to watch with respect to the directory
	PathRegexp string `json:"pathRegexp,omitempty"`
}

// EventSource refers to a dependency on a k8s resource.
type ResourceEventSource struct {
	// Namespace where resource is deployed
	Namespace string `json:"namespace"`
	// Filter is applied on the metadata of the resource
	Filter *ResourceFilter `json:"filter,omitempty"`
	// Version of the source
	Version string `json:"version"`
	// Group of the resource
	metav1.GroupVersionKind `json:",inline"`
	// Type is the event type. Refer https://github.com/kubernetes/apimachinery/blob/dcb391cde5ca0298013d43336817d20b74650702/pkg/watch/watch.go#L43
	// If not provided, the gateway will watch all events for a resource.
	Type watch.EventType `json:"type,omitempty"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource event objects
type ResourceFilter struct {
	Prefix      string            `json:"prefix,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedBy   metav1.Time       `json:"createdBy,omitempty"`
}

// AMQPEventSource contains configuration required to connect to rabbitmq service and process messages
type AMQPEventSource struct {
	// URL for rabbitmq service
	URL string `json:"url"`
	// ExchangeName is the exchange name
	// For more information, visit https://www.rabbitmq.com/tutorials/amqp-concepts.html
	ExchangeName string `json:"exchangeName"`
	// ExchangeType is rabbitmq exchange type
	ExchangeType string `json:"exchangeType"`
	// Routing key for bindings
	RoutingKey string `json:"routingKey"`
	// Backoff holds parameters applied to connection.
	Backoff *wait.Backoff `json:"backoff,omitempty"`
}

// KafkaEventSource defines configuration required to connect to kafka cluster
type KafkaEventSource struct {
	// URL to kafka cluster
	URL string `json:"url"`
	// Partition name
	Partition string `json:"partition"`
	// Topic name
	Topic string `json:"topic"`
	// Backoff holds parameters applied to connection.
	Backoff *wait.Backoff `json:"backoff,omitempty"`
}

// MQTTEventSource contains information to connect to MQTT broker
type MQTTEventSource struct {
	// URL to connect to broker
	URL string `json:"url"`
	// Topic name
	Topic string `json:"topic"`
	// Client ID
	ClientId string `json:"clientId"`
	// Backoff holds parameters applied to connection.
	Backoff *wait.Backoff `json:"backoff,omitempty"`
}

// NATSEventSource contains configuration to connect to NATS cluster
type NATSEventSource struct {
	// URL to connect to natsConfig cluster
	URL string `json:"url"`
	// Subject name
	Subject string `json:"subject"`
	// Backoff holds parameters applied to connection.
	Backoff *wait.Backoff `json:"backoff,omitempty"`
}

// SNSEventSource contains configuration to subscribe to SNS topic
type SNSEventSource struct {
	// Hook defines a webhook.
	Hook *WebhookEventSource `json:"hook"`
	// TopicArn to connect to
	TopicArn string `json:"topicArn"`
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey" protobuf:"bytes,5,opt,name=accessKey"`
	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey" protobuf:"bytes,6,opt,name=secretKey"`
	// Region to operate in
	Region string `json:"region"`
}

// SQSEventSource contains information to listen to AWS SQS
type SQSEventSource struct {
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey"`
	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey"`
	// Region to operate in
	Region string `json:"region"`
	// Queue is AWS SQS queue to listen to for messages
	Queue string `json:"queue"`
	// WaitTimeSeconds is The duration (in seconds) for which the call waits for a message to arrive
	// in the queue before returning.
	WaitTimeSeconds int64 `json:"waitTimeSeconds"`
}

// PubSubEventSource contains configuration to subscribe to GCP PubSub topic
type PubSubEventSource struct {
	// ProjectID is the unique identifier for your project on GCP
	ProjectID string `json:"projectID"`
	// Topic on which a subscription will be created
	Topic string `json:"topic"`
	// CredentialsFile is the file that contains credentials to authenticate for GCP
	CredentialsFile string `json:"credentialsFile"`
}

// GithubEventSource contains information to setup a github project integration
type GithubEventSource struct {
	// Webhook ID
	Id int64 `json:"id"`
	// Webhook
	Hook *common.Webhook `json:"hook"`
	// GitHub owner name i.e. argoproj
	Owner string `json:"owner"`
	// GitHub repo name i.e. argo-events
	Repository string `json:"repository"`
	// Github events to subscribe to which the gateway will subscribe
	Events []string `json:"events"`
	// K8s secret containing github api token
	APIToken *corev1.SecretKeySelector `json:"apiToken"`
	// K8s secret containing WebHook Secret
	WebHookSecret *corev1.SecretKeySelector `json:"webHookSecret"`
	// Insecure tls verification
	Insecure bool `json:"insecure"`
	// Active
	Active bool `json:"active"`
	// ContentType json or form
	ContentType string `json:"contentType"`
	// GitHub base URL (for GitHub Enterprise)
	GithubBaseURL string `json:"githubBaseURL"`
	// GitHub upload URL (for GitHub Enterprise)
	GithubUploadURL string `json:"githubUploadURL"`
}

// GitlabEventSource contains information to setup a gitlab project integration
type GitlabEventSource struct {
	// Webhook Id
	Id int `json:"id"`
	// Webhook
	Hook *common.Webhook `json:"hook"`
	// ProjectId is the id of project for which integration needs to setup
	ProjectId string `json:"projectId"`
	// Event is a gitlab event to listen to.
	// Refer https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794.
	Event string `json:"event"`
	// AccessToken is reference to k8 secret which holds the gitlab api access information
	AccessToken *corev1.SecretKeySelector `json:"accessToken"`
	// EnableSSLVerification to enable ssl verification
	EnableSSLVerification bool `json:"enableSSLVerification"`
	// GitlabBaseURL is the base URL for API requests to a custom endpoint
	GitlabBaseURL string `json:"gitlabBaseURL"`
}

// HDFSEventSource contains information to setup a HDFS integration
type HDFSEventSource struct {
	WatchPathConfig `json:",inline"`
	// Type of file operations to watch
	Type string `json:"type"`
	// CheckInterval is a string that describes an interval duration to check the directory state, e.g. 1s, 30m, 2h... (defaults to 1m)
	CheckInterval    string `json:"checkInterval,omitempty"`
	HDFSClientConfig `json:",inline"`
}

// HDFSClientConfig contains HDFS client configurations
type HDFSClientConfig struct {
	// Addresses is accessible addresses of HDFS name nodes
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
}

// SlackEventSource contains configuration for slack
type SlackEventSource struct {
	// Token for URL verification handshake
	Token *corev1.SecretKeySelector `json:"token"`
	// Webhook
	Hook *common.Webhook `json:"hook"`
}

// StorageGridEventSource contains configuration for storage grid sns
type StorageGridEventSource struct {
	// Webhook
	Hook *WebhookEventSource `json:"hook"`
	// Events are s3 bucket notification events.
	// For more information on s3 notifications, follow https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations
	// Note that storage grid notifications do not contain `s3:`
	Events []string `json:"events,omitempty"`
	// Filter on object key which caused the notification.
	Filter *Filter `json:"filter,omitempty"`
}

// Filter represents filters to apply to bucket notifications for specifying constraints on objects
type Filter struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}
