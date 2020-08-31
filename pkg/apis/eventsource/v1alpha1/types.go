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
// +kubebuilder:resource:shortName=es
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type EventSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              EventSourceSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            EventSourceStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// EventSourceList is the list of eventsource resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EventSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Items []EventSource `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// EventSourceSpec refers to specification of event-source resource
type EventSourceSpec struct {
	// EventBusName references to a EventBus name. By default the value is "default"
	EventBusName string `json:"eventBusName,omitempty" protobuf:"bytes,1,opt,name=eventBusName"`
	// Template is the pod specification for the event source
	// +optional
	Template Template `json:"template,omitempty" protobuf:"bytes,2,opt,name=template"`
	// Service is the specifications of the service to expose the event source
	// +optional
	Service *Service `json:"service,omitempty" protobuf:"bytes,3,opt,name=service"`
	// Replica is the event source deployment replicas
	Replica *int32 `json:"replica,omitempty" protobuf:"varint,4,opt,name=replica"`

	// Minio event sources
	Minio map[string]apicommon.S3Artifact `json:"minio,omitempty" protobuf:"bytes,5,rep,name=minio"`
	// Calendar event sources
	Calendar map[string]CalendarEventSource `json:"calendar,omitempty" protobuf:"bytes,6,rep,name=calendar"`
	// File event sources
	File map[string]FileEventSource `json:"file,omitempty" protobuf:"bytes,7,rep,name=file"`
	// Resource event sources
	Resource map[string]ResourceEventSource `json:"resource,omitempty" protobuf:"bytes,8,rep,name=resource"`
	// Webhook event sources
	Webhook map[string]WebhookContext `json:"webhook,omitempty" protobuf:"bytes,9,rep,name=webhook"`
	// AMQP event sources
	AMQP map[string]AMQPEventSource `json:"amqp,omitempty" protobuf:"bytes,10,rep,name=amqp"`
	// Kafka event sources
	Kafka map[string]KafkaEventSource `json:"kafka,omitempty" protobuf:"bytes,11,rep,name=kafka"`
	// MQTT event sources
	MQTT map[string]MQTTEventSource `json:"mqtt,omitempty" protobuf:"bytes,12,rep,name=mqtt"`
	// NATS event sources
	NATS map[string]NATSEventsSource `json:"nats,omitempty" protobuf:"bytes,13,rep,name=nats"`
	// SNS event sources
	SNS map[string]SNSEventSource `json:"sns,omitempty" protobuf:"bytes,14,rep,name=sns"`
	// SQS event sources
	SQS map[string]SQSEventSource `json:"sqs,omitempty" protobuf:"bytes,15,rep,name=sqs"`
	// PubSub eevnt sources
	PubSub map[string]PubSubEventSource `json:"pubSub,omitempty" protobuf:"bytes,16,rep,name=pubSub"`
	// Github event sources
	Github map[string]GithubEventSource `json:"github,omitempty" protobuf:"bytes,17,rep,name=github"`
	// Gitlab event sources
	Gitlab map[string]GitlabEventSource `json:"gitlab,omitempty" protobuf:"bytes,18,rep,name=gitlab"`
	// HDFS event sources
	HDFS map[string]HDFSEventSource `json:"hdfs,omitempty" protobuf:"bytes,19,rep,name=hdfs"`
	// Slack event sources
	Slack map[string]SlackEventSource `json:"slack,omitempty" protobuf:"bytes,20,rep,name=slack"`
	// StorageGrid event sources
	StorageGrid map[string]StorageGridEventSource `json:"storageGrid,omitempty" protobuf:"bytes,21,rep,name=storageGrid"`
	// AzureEventsHub event sources
	AzureEventsHub map[string]AzureEventsHubEventSource `json:"azureEventsHub,omitempty" protobuf:"bytes,22,rep,name=azureEventsHub"`
	// Stripe event sources
	Stripe map[string]StripeEventSource `json:"stripe,omitempty" protobuf:"bytes,23,rep,name=stripe"`
	// Emitter event source
	Emitter map[string]EmitterEventSource `json:"emitter,omitempty" protobuf:"bytes,24,rep,name=emitter"`
	// Redis event source
	Redis map[string]RedisEventSource `json:"redis,omitempty" protobuf:"bytes,25,rep,name=redis"`
	// NSQ event source
	NSQ map[string]NSQEventSource `json:"nsq,omitempty" protobuf:"bytes,26,rep,name=nsq"`
	// Pulsar event source
	Pulsar map[string]PulsarEventSource `json:"pulsar,omitempty" protobuf:"bytes,27,opt,name=pulsar"`
	// Generic event source
	Generic map[string]GenericEventSource `json:"generic,omitempty" protobuf:"bytes,28,rep,name=generic"`
}

// Template holds the information of an EventSource deployment template
type Template struct {
	// Metdata sets the pods's metadata, i.e. annotations and labels
	Metadata Metadata `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// ServiceAccountName is the name of the ServiceAccount to use to run event source pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,2,opt,name=serviceAccountName"`
	// Container is the main container image to run in the event source pod
	// +optional
	Container *corev1.Container `json:"container,omitempty" protobuf:"bytes,3,opt,name=container"`
	// Volumes is a list of volumes that can be mounted by containers in a workflow.
	// +patchStrategy=merge
	// +patchMergeKey=name
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,4,rep,name=volumes"`
	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,5,opt,name=securityContext"`
	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty" protobuf:"bytes,6,opt,name=affinity"`
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty" protobuf:"bytes,7,rep,name=tolerations"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,8,rep,name=nodeSelector"`
}

// Metadata holds the annotations and labels of an event source pod
type Metadata struct {
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,1,rep,name=annotations"`
	Labels      map[string]string `json:"labels,omitempty" protobuf:"bytes,2,rep,name=labels"`
}

// Service holds the service information eventsource exposes
type Service struct {
	// The list of ports that are exposed by this ClusterIP service.
	// +patchMergeKey=port
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=port
	// +listMapKey=protocol
	Ports []corev1.ServicePort `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"port" protobuf:"bytes,1,rep,name=ports"`
	// clusterIP is the IP address of the service and is usually assigned
	// randomly by the master. If an address is specified manually and is not in
	// use by others, it will be allocated to the service; otherwise, creation
	// of the service will fail. This field can not be changed through updates.
	// Valid values are "None", empty string (""), or a valid IP address. "None"
	// can be specified for headless services when proxying is not required.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies
	// +optional
	ClusterIP string `json:"clusterIP,omitempty" protobuf:"bytes,2,opt,name=clusterIP"`
}

// CalendarEventSource describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type CalendarEventSource struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule" protobuf:"bytes,1,opt,name=schedule"`
	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval" protobuf:"bytes,2,opt,name=interval"`
	// ExclusionDates defines the list of DATE-TIME exceptions for recurring events.

	ExclusionDates []string `json:"exclusionDates,omitempty" protobuf:"bytes,3,rep,name=exclusionDates"`
	// Timezone in which to run the schedule
	// +optional
	Timezone string `json:"timezone,omitempty" protobuf:"bytes,4,opt,name=timezone"`
	// UserPayload will be sent to sensor as extra data once the event is triggered
	// +optional
	// Deprecated. Please use Metadata instead.
	UserPayload json.RawMessage `json:"userPayload,omitempty" protobuf:"bytes,5,opt,name=userPayload,casttype=encoding/json.RawMessage"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,6,rep,name=metadata"`
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
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,4,rep,name=metadata"`
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
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,5,rep,name=metadata"`
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
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,7,opt,name=tls"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,8,rep,name=metadata"`
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
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,5,opt,name=tls"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,6,opt,name=jsonBody"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,7,rep,name=metadata"`

	// Consumer group for kafka client
	// +optional
	ConsumerGroup *KafkaConsumerGroup `json:"consumerGroup,omitempty" protobuf:"bytes,8,opt,name=consumerGroup"`

	// Sets a limit on how many events get read from kafka per second.
	// +optional
	LimitEventsPerSecond int64 `json:"limitEventsPerSecond,omitempty" protobuf:"varint,9,opt,name=limitEventsPerSecond"`

	// Specify what kafka version is being connected to enables certain features in sarama, defaults to 1.0.0
	// +optional
	Version string `json:"version" protobuf:"bytes,10,opt,name=version"`
}

type KafkaConsumerGroup struct {
	// The name for the consumer group to use
	GroupName string `json:"groupName" protobuf:"bytes,1,opt,name=groupName"`
	// When starting up a new group do we want to start from the oldest event (true) or the newest event (false), defaults to false
	// +optional
	Oldest bool `json:"oldest,omitempty" protobuf:"varint,2,opt,name=oldest"`
	// Rebalance strategy can be one of: sticky, roundrobin, range. Range is the default.
	// +optional
	RebalanceStrategy string `json:"rebalanceStrategy" protobuf:"bytes,3,opt,name=rebalanceStrategy"`
}

// MQTTEventSource refers to event-source for MQTT related events
type MQTTEventSource struct {
	// URL to connect to broker
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Topic name
	Topic string `json:"topic" protobuf:"bytes,2,opt,name=topic"`
	// ClientID is the id of the client
	ClientID string `json:"clientId" protobuf:"bytes,3,opt,name=clientId"`
	// ConnectionBackoff holds backoff applied to connection.
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,4,opt,name=connectionBackoff"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,5,opt,name=jsonBody"`
	// TLS configuration for the mqtt client.
	// +optional
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,6,opt,name=tls"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,7,rep,name=metadata"`
}

// NATSEventsSource refers to event-source for NATS related events
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
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,5,opt,name=tls"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,6,rep,name=metadata"`
}

// SNSEventSource refers to event-source for AWS SNS related events
type SNSEventSource struct {
	// Webhook configuration for http server
	Webhook *WebhookContext `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`
	// TopicArn
	TopicArn string `json:"topicArn" protobuf:"bytes,2,opt,name=topicArn"`
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,3,opt,name=accessKey"`
	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,4,opt,name=secretKey"`
	// Region is AWS region
	Region string `json:"region" protobuf:"bytes,5,opt,name=region"`
	// RoleARN is the Amazon Resource Name (ARN) of the role to assume.
	// +optional
	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,6,opt,name=roleARN"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,7,rep,name=metadata"`
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
	// RoleARN is the Amazon Resource Name (ARN) of the role to assume.
	// +optional
	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,6,opt,name=roleARN"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,7,opt,name=jsonBody"`
	// QueueAccountID is the ID of the account that created the queue to monitor
	// +optional
	QueueAccountID string `json:"queueAccountId,omitempty" protobuf:"bytes,8,opt,name=queueAccountId"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,9,rep,name=metadata"`
}

// PubSubEventSource refers to event-source for GCP PubSub related events.
type PubSubEventSource struct {
	// ProjectID is GCP project ID for the subscription.
	// Required if you run Argo Events outside of GKE/GCE.
	// (otherwise, the default value is its project)
	// +optional
	ProjectID string `json:"projectID" protobuf:"bytes,1,opt,name=projectID"`
	// TopicProjectID is GCP project ID for the the topic.
	// By default, it is same as ProjectID.
	// +optional
	TopicProjectID string `json:"topicProjectID" protobuf:"bytes,2,opt,name=topicProjectID"`
	// Topic to which the subscription should belongs.
	// Required if you want the eventsource to create a new subscription.
	// If you specify this field along with an existing subscription,
	// it will be verified whether it actually belongs to the specified topic.
	// +optional
	Topic string `json:"topic" protobuf:"bytes,3,opt,name=topic"`
	// SubscriptionID is ID of subscription.
	// Required if you use existing subscription.
	// The default value will be auto generated hash based on this eventsource setting, so the subscription
	// might be recreated every time you update the setting, which has a possiblity of event loss.
	// +optional
	SubscriptionID string `json:"subscriptionID" protobuf:"bytes,4,opt,name=subscriptionID"`
	// CredentialSecret references to the secret that contains JSON credentials to access GCP.
	// If it is missing, it implicts to use Workload Identity to access.
	// https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
	// +optional
	CredentialSecret *corev1.SecretKeySelector `json:"credentialSecret,omitempty" protobuf:"bytes,5,opt,name=credentialSecret"`
	// DeleteSubscriptionOnFinish determines whether to delete the GCP PubSub subscription once the event source is stopped.
	// +optional
	DeleteSubscriptionOnFinish bool `json:"deleteSubscriptionOnFinish,omitempty" protobuf:"varint,6,opt,name=deleteSubscriptionOnFinish"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,7,opt,name=jsonBody"`
	// CredentialsFile is the file that contains credentials to authenticate for GCP
	// Deprecated, use CredentialSecret instead
	DeprecatedCredentialsFile string `json:"credentialsFile" protobuf:"bytes,8,opt,name=credentialsFile"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,9,rep,name=metadata"`
}

// GithubEventSource refers to event-source for github related events
type GithubEventSource struct {
	// Id is the webhook's id
	ID int64 `json:"id" protobuf:"varint,1,opt,name=id"`
	// Webhook refers to the configuration required to run a http server
	Webhook *WebhookContext `json:"webhook,omitempty" protobuf:"bytes,2,opt,name=webhook"`
	// Owner refers to GitHub owner name i.e. argoproj
	Owner string `json:"owner" protobuf:"bytes,3,opt,name=owner"`
	// Repository refers to GitHub repo name i.e. argo-events
	Repository string `json:"repository" protobuf:"bytes,4,opt,name=repository"`
	// Events refer to Github events to subscribe to which the event source will subscribe

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
	// DeleteHookOnFinish determines whether to delete the GitHub hook for the repository once the event source is stopped.
	// +optional
	DeleteHookOnFinish bool `json:"deleteHookOnFinish,omitempty" protobuf:"varint,13,opt,name=deleteHookOnFinish"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,14,rep,name=metadata"`
}

// GitlabEventSource refers to event-source related to Gitlab events
type GitlabEventSource struct {
	// Webhook holds configuration to run a http server
	Webhook *WebhookContext `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`
	// ProjectID is the id of project for which integration needs to setup
	ProjectID string `json:"projectID" protobuf:"bytes,2,opt,name=projectID"`
	// Events are gitlab event to listen to.
	// Refer https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794.
	Events []string `json:"events" protobuf:"bytes,3,opt,name=events"`
	// AccessToken is reference to k8 secret which holds the gitlab api access information
	AccessToken *corev1.SecretKeySelector `json:"accessToken,omitempty" protobuf:"bytes,4,opt,name=accessToken"`
	// EnableSSLVerification to enable ssl verification
	// +optional
	EnableSSLVerification bool `json:"enableSSLVerification,omitempty" protobuf:"varint,5,opt,name=enableSSLVerification"`
	// GitlabBaseURL is the base URL for API requests to a custom endpoint
	GitlabBaseURL string `json:"gitlabBaseURL" protobuf:"bytes,6,opt,name=gitlabBaseURL"`
	// DeleteHookOnFinish determines whether to delete the GitLab hook for the project once the event source is stopped.
	// +optional
	DeleteHookOnFinish bool `json:"deleteHookOnFinish,omitempty" protobuf:"varint,8,opt,name=deleteHookOnFinish"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,9,rep,name=metadata"`
}

// HDFSEventSource refers to event-source for HDFS related events
type HDFSEventSource struct {
	WatchPathConfig `json:",inline" protobuf:"bytes,1,opt,name=watchPathConfig"`
	// Type of file operations to watch
	Type string `json:"type" protobuf:"bytes,2,opt,name=type"`
	// CheckInterval is a string that describes an interval duration to check the directory state, e.g. 1s, 30m, 2h... (defaults to 1m)
	CheckInterval string `json:"checkInterval,omitempty" protobuf:"bytes,3,opt,name=checkInterval"`
	// Addresses is accessible addresses of HDFS name nodes

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
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,12,rep,name=metadata"`
}

// SlackEventSource refers to event-source for Slack related events
type SlackEventSource struct {
	// Slack App signing secret
	SigningSecret *corev1.SecretKeySelector `json:"signingSecret,omitempty" protobuf:"bytes,1,opt,name=signingSecret"`
	// Token for URL verification handshake
	Token *corev1.SecretKeySelector `json:"token,omitempty" protobuf:"bytes,2,opt,name=token"`
	// Webhook holds configuration for a REST endpoint
	Webhook *WebhookContext `json:"webhook,omitempty" protobuf:"bytes,3,opt,name=webhook"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,4,rep,name=metadata"`
}

// StorageGridEventSource refers to event-source for StorageGrid related events
type StorageGridEventSource struct {
	// Webhook holds configuration for a REST endpoint
	Webhook *WebhookContext `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`
	// Events are s3 bucket notification events.
	// For more information on s3 notifications, follow https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations
	// Note that storage grid notifications do not contain `s3:`

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
	// APIURL is the url of the storagegrid api.
	APIURL string `json:"apiURL" protobuf:"bytes,8,name=apiURL"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,9,rep,name=metadata"`
}

// StorageGridFilter represents filters to apply to bucket notifications for specifying constraints on objects
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
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,5,rep,name=metadata"`
}

// StripeEventSource describes the event source for stripe webhook notifications
// More info at https://stripe.com/docs/webhooks
type StripeEventSource struct {
	// Webhook holds configuration for a REST endpoint
	Webhook *WebhookContext `json:"webhook,omitempty" protobuf:"bytes,1,opt,name=webhook"`
	// CreateWebhook if specified creates a new webhook programmatically.
	// +optional
	CreateWebhook bool `json:"createWebhook,omitempty" protobuf:"varint,2,opt,name=createWebhook"`
	// APIKey refers to K8s secret that holds Stripe API key. Used only if CreateWebhook is enabled.
	// +optional
	APIKey *corev1.SecretKeySelector `json:"apiKey,omitempty" protobuf:"bytes,3,opt,name=apiKey"`
	// EventFilter describes the type of events to listen to. If not specified, all types of events will be processed.
	// More info at https://stripe.com/docs/api/events/list
	// +optional
	EventFilter []string `json:"eventFilter,omitempty" protobuf:"bytes,4,rep,name=eventFilter"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,5,rep,name=metadata"`
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
	// Username to use to connect to broker
	// +optional
	Username *corev1.SecretKeySelector `json:"username,omitempty" protobuf:"bytes,4,opt,name=username"`
	// Password to use to connect to broker
	// +optional
	Password *corev1.SecretKeySelector `json:"password,omitempty" protobuf:"bytes,5,opt,name=password"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,6,opt,name=connectionBackoff"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"varint,7,opt,name=jsonBody"`
	// TLS configuration for the emitter client.
	// +optional
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,8,opt,name=tls"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,9,rep,name=metadata"`
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

	Channels []string `json:"channels" protobuf:"bytes,5,rep,name=channels"`
	// TLS configuration for the redis client.
	// +optional
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,6,opt,name=tls"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,7,rep,name=metadata"`
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
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,6,opt,name=tls"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,7,rep,name=metadata"`
}

// PulsarEventSource describes the event source for Apache Pulsar
type PulsarEventSource struct {
	// Name of the topics to subscribe to.
	// +required
	Topics []string `json:"topics" protobuf:"bytes,1,rep,name=topics"`
	// Type of the subscription.
	// Only "exclusive" and "shared" is supported.
	// Defaults to exclusive.
	// +optional
	Type string `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`
	// Configure the service URL for the Pulsar service.
	// +required
	URL string `json:"url" protobuf:"bytes,3,name=url"`
	// Trusted TLS certificate secret.
	// +optional
	TLSTrustCertsSecret *corev1.SecretKeySelector `json:"tlsTrustCertsSecret,omitempty" protobuf:"bytes,4,opt,name=tlsTrustCertsSecret"`
	// Whether the Pulsar client accept untrusted TLS certificate from broker.
	// +optional
	TLSAllowInsecureConnection bool `json:"tlsAllowInsecureConnection,omitempty" protobuf:"bytes,5,opt,name=tlsAllowInsecureConnection"`
	// Whether the Pulsar client verify the validity of the host name from broker.
	// +optional
	TLSValidateHostname bool `json:"tlsValidateHostname,omitempty" protobuf:"bytes,6,opt,name=tlsValidateHostname"`
	// TLS configuration for the pulsar client.
	// +optional
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,7,opt,name=tls"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *apicommon.Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,8,opt,name=connectionBackoff"`
	// JSONBody specifies that all event body payload coming from this
	// source will be JSON
	// +optional
	JSONBody bool `json:"jsonBody,omitempty" protobuf:"bytes,9,opt,name=jsonBody"`
	// Metadata holds the user defined metadata which will passed along the event payload.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,10,rep,name=metadata"`
}

// GenericEventSource refers to a generic event source. It can be used to implement a custom event source.
type GenericEventSource struct {
	// Value of the event source
	Value string `json:"value" protobuf:"bytes,1,opt,name=value"`
}

const (
	// EventSourceConditionSourcesProvided has the status True when the EventSource
	// has its event source provided.
	EventSourceConditionSourcesProvided apicommon.ConditionType = "SourcesProvided"
	// EventSourceConditionDeployed has the status True when the EventSource
	// has its Deployment created.
	EventSourceConditionDeployed apicommon.ConditionType = "Deployed"
)

// EventSourceStatus holds the status of the event-source resource
type EventSourceStatus struct {
	apicommon.Status `json:",inline" protobuf:"bytes,1,opt,name=status"`
}

// InitConditions sets conditions to Unknown state.
func (es *EventSourceStatus) InitConditions() {
	es.InitializeConditions(EventSourceConditionSourcesProvided, EventSourceConditionDeployed)
}

// MarkSourcesProvided set the eventsource has valid sources spec provided.
func (es *EventSourceStatus) MarkSourcesProvided() {
	es.MarkTrue(EventSourceConditionSourcesProvided)
}

// MarkSourcesNotProvided the eventsource has invalid sources spec provided.
func (es *EventSourceStatus) MarkSourcesNotProvided(reason, message string) {
	es.MarkFalse(EventSourceConditionSourcesProvided, reason, message)
}

// MarkDeployed set the eventsource has been deployed.
func (es *EventSourceStatus) MarkDeployed() {
	es.MarkTrue(EventSourceConditionDeployed)
}

// MarkDeployFailed set the eventsource deploy failed
func (es *EventSourceStatus) MarkDeployFailed(reason, message string) {
	es.MarkFalse(EventSourceConditionDeployed, reason, message)
}
