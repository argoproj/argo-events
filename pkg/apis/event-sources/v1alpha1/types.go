package v1alpha1

import (
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
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
	Spec              EventSourceSpec   `json:"spec" protobuf:"bytes,3,opt,name=spec"`
}

// EventSourceList is the list of eventsource resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EventSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	// +listType=items
	Items []EventSource `json:"items" protobuf:"bytes,2,opt,name=items"`
}

// EventSourceSpec refers to specification of event-source resource
type EventSourceSpec struct {
	Minio map[string]apicommon.S3Artifact `json:"minio,omitempty" protobuf:"bytes,1,opt,name=minio"`

	Calendar map[string]CalendarEventSource `json:"calendar,omitempty" protobuf:"bytes,2,opt,name=calendar"`

	File map[string]FileEventSource `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`

	Resource map[string]ResourceEventSource `json:"resource,omitempty" protobuf:"bytes,4,opt,name=resource"`

	Webhook map[string]gwcommon.Webhook `json:"webhook,omitempty" protobuf:"bytes,5,opt,name=webhook"`

	AMQP map[string]AMQPEventSource `json:"amqp,omitempty" protobuf:"bytes,6,opt,name=amqp"`

	Kafka map[string]KafkaEventSource `json:"kafka,omitempty" protobuf:"bytes,7,opt,name=kafka"`

	MQTT map[string]MQTTEventSource `json:"mqtt,omitempty" protobuf:"bytes,8,opt,name=mqtt"`

	NATS map[string]NATSEventsSource `json:"nats,omitempty" protobuf:"bytes,9,opt,name=nats"`
}

// CalendarEventSource describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type CalendarEventSource struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule"`
	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval"`
	// ExclusionDates defines the list of DATE-TIME exceptions for recurring events.
	ExclusionDates []string `json:"recurrence,omitempty"`
	// Timezone in which to run the schedule
	// +optional
	Timezone string `json:"timezone,omitempty"`
	// UserPayload will be sent to sensor as extra data once the event is triggered
	// +optional
	UserPayload *json.RawMessage `json:"userPayload,omitempty"`
}

// FileEventSource describes an event-source for file related events.
type FileEventSource struct {
	// Directory to watch for events
	Directory string `json:"directory" protobuf:"bytes,1,name=directory"`
	// Path is relative path of object to watch with respect to the directory
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`
	// PathRegexp is regexp of relative path of object to watch with respect to the directory
	// +optional
	PathRegexp string `json:"pathRegexp,omitempty" protobuf:"bytes,3,opt,name=pathRegexp"`
	// Type of file operations to watch
	// Refer https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go for more information
	EventType string `json:"eventType" protobuf:"bytes,4,name=eventType"`
}

type EventType string

const (
	ADD    EventType = "ADD"
	UPDATE EventType = "UPDATE"
	DELETE EventType = "DELETE"
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
	EventType EventType `json:"eventType,omitempty" protobuf:"bytes,3,opt,name=eventType"`
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
	ConnectionBackoff *common.Backoff `json:"backoff,omitempty" protobuf:"bytes,4,opt,name=connectionBackoff"`
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
	// WebHook configuration for http server
	WebHook *gwcommon.Webhook `json:"hook"`
	// TopicArn
	TopicArn string `json:"topicArn"`
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,5,opt,name=accessKey"`
	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,6,opt,name=secretKey"`
	// Region is AWS region
	Region string `json:"region"`
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
}

// GithubEventSource refers to event-source for github related events
type GithubEventSource struct {
	// Webhook ID
	Id int64 `json:"id"`
	// Webhook
	Hook *gwcommon.Webhook `json:"hook"`
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

// EventSourceStatus holds the status of the event-source resource
type EventSourceStatus struct {
	CreatedAt metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,1,opt,name=createdAt"`
}
