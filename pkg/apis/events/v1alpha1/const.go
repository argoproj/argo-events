package v1alpha1

import (
	"fmt"
	"reflect"

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
	TriggerTypeOpenWhisk       TriggerType = "OpenWhisk"
	TriggerTypeArgoWorkflow    TriggerType = "ArgoWorkflow"
	TriggerTypeLambda          TriggerType = "Lambda"
	TriggerTypeCustom          TriggerType = "Custom"
	TriggerTypeHTTP            TriggerType = "HTTP"
	TriggerTypeKafka           TriggerType = "Kafka"
	TriggerTypePulsar          TriggerType = "Pulsar"
	TriggerTypeLog             TriggerType = "Log"
	TriggerTypeNATS            TriggerType = "NATS"
	TriggerTypeSlack           TriggerType = "Slack"
	TriggerTypeK8s             TriggerType = "Kubernetes"
	TriggerTypeAzureEventHubs  TriggerType = "AzureEventHubs"
	TriggerTypeAzureServiceBus TriggerType = "AzureServiceBus"
	TriggerTypeEmail           TriggerType = "Email"
)

// EventBusType is the type of event bus
type EventBusType string

// possible event bus types
var (
	EventBusNATS      EventBusType = "nats"
	EventBusJetStream EventBusType = "jetstream"
	EventBusKafka     EventBusType = "kafka"
)

// Environment variables
const (
	// EnvVarKubeConfig is the path to the Kubernetes configuration
	EnvVarKubeConfig = "KUBECONFIG"
	// ENVVarPodName should be set to the name of the pod
	EnvVarPodName = "POD_NAME"
	// ENVVarLeaderElection sets the leader election mode
	EnvVarLeaderElection = "LEADER_ELECTION"
	// EnvImagePullPolicy is the env var to set container's ImagePullPolicy
	EnvImagePullPolicy = "IMAGE_PULL_POLICY"
)

// EventBus related
const (
	// EnvVarEventBusConfig refers to the eventbus config env
	EnvVarEventBusConfig = "EVENTBUS_CONFIG"
	// EnvVarEventBusSubject refers to the eventbus subject env
	EnvVarEventBusSubject = "EVENTBUS_SUBJECT"
	// volumeMount path for eventbus auth file
	EventBusAuthFileMountPath = "/etc/eventbus/auth"
	// Default NATS Streaming messages max age
	STANMaxAge = "72h"
	// Default NATS Streaming max messages per channel
	STANMaxMsgs = uint64(1000000)
	// Default NATS Streaming max subscriptions per channel
	STANMaxSubs = uint64(1000)
	// Default NATS Streaming max total size of messages per channel
	STANMaxBytes = "1GB"
	// Default NATS Streaming max size of message payload
	STANMaxPayload = "1MB"
	// Default NATS Streaming RAFT heartbeat timeout
	STANRaftHeartbeatTimeout = "2s"
	// Default NATS Streaming RAFT election timeout
	STANRaftElectionTimeout = "2s"
	// Default NATS Streaming RAFT lease timeout
	STANRaftLeaseTimeout = "1s"
	// Default NATS Streaming RAFT commit timeout
	STANRaftCommitTimeout = "100ms"

	// Default EventBus name
	DefaultEventBusName = "default"

	// key of auth server secret
	JetStreamServerSecretAuthKey = "auth"
	// key of encryption server secret
	JetStreamServerSecretEncryptionKey = "encryption"
	// key of client auth secret
	JetStreamClientAuthSecretKey = "client-auth"
	// key for server private key
	JetStreamServerPrivateKeyKey = "private-key"
	// key for server TLS certificate
	JetStreamServerCertKey = "cert"
	// key for server CA certificate
	JetStreamServerCACertKey = "ca-cert"
	// key for server private key
	JetStreamClusterPrivateKeyKey = "cluster-private-key"
	// key for server TLS certificate
	JetStreamClusterCertKey = "cluster-cert"
	// key for server CA certificate
	JetStreamClusterCACertKey = "cluster-ca-cert"
	// key of nats-js.conf in the configmap
	JetStreamConfigMapKey = "nats-js"
	// Jetstream Stream name
	JetStreamStreamName = "default"
	// Default JetStream max size of message payload
	JetStreamMaxPayload = "1MB"
)

// Sensor constants
const (
	// EnvVarSensorObject refers to the env of based64 encoded sensor spec
	EnvVarSensorObject = "SENSOR_OBJECT"
	// SensorNamespace is used to get namespace where sensors are deployed
	SensorNamespace = "SENSOR_NAMESPACE"
	// LabelSensorName is label for sensor name
	LabelSensorName = "sensor-name"
)

// EventSource
const (
	// EnvVarEventSourceObject refers to the env of based64 encoded eventsource spec
	EnvVarEventSourceObject = "EVENTSOURCE_OBJECT"
	// EnvVarEventSource refers to event source name
	EnvVarEventSource = "EVENT_SOURCE"
	// LabelEventSourceName is the label for a event source
	LabelEventSourceName = "eventsource-name"
)

var (
	ErrNilEventSource = fmt.Errorf("event source can't be nil")
)

// Miscellaneous Labels
const (
	// LabelOwnerName is the label for resource owner name
	LabelOwnerName = "owner-name"
	// AnnotationResourceSpecHash is the annotation of a K8s resource spec hash
	AnnotationResourceSpecHash = "resource-spec-hash"
	// AnnotationLeaderElection is the annotation for leader election
	AnnotationLeaderElection = "events.argoproj.io/leader-election"
)

// various supported media types
const (
	MediaTypeJSON string = "application/json"
	MediaTypeYAML string = "application/yaml"
)

// Metrics releated
const (
	EventSourceMetricsPort = 7777
	SensorMetricsPort      = 7777
	ControllerMetricsPort  = 7777
	EventBusMetricsPort    = 7777
	ControllerHealthPort   = 8081
)

var (
	SecretKeySelectorType    = reflect.TypeOf(&corev1.SecretKeySelector{})
	ConfigMapKeySelectorType = reflect.TypeOf(&corev1.ConfigMapKeySelector{})
)
