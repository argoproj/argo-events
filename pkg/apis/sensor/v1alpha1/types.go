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
	"fmt"
	"hash/fnv"

	"github.com/minio/minio-go"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SignalType is the type of the signal
type SignalType string

// possible types of signals or inputs
const (
	SignalTypeAMQP     SignalType = "AMQP"
	SignalTypeMQTT     SignalType = "MQTT"
	SignalTypeNats     SignalType = "NATS"
	SignalTypeKafka    SignalType = "Kafka"
	SignalTypeArtifact SignalType = "Artifact"
	SignalTypeCalendar SignalType = "Calendar"
	SignalTypeResource SignalType = "Resource"
	SignalTypeWebhook  SignalType = "Webhook"
)

// StreamType is the type of a stream
type StreamType string

// possible types of streams
const (
	StreamTypeAMQP  StreamType = "AMQP"
	StreamTypeMQTT  StreamType = "MQTT"
	StreamTypeNats  StreamType = "NATS"
	StreamTypeKafka StreamType = "Kafka"
)

// NodeType is the type of a node
type NodeType string

// possible node types
const (
	NodeTypeSignal  NodeType = "Signal"
	NodeTypeTrigger NodeType = "Trigger"
)

// NodePhase is the label for the condition of a node
type NodePhase string

// possible types of node phases
const (
	NodePhaseSucceeded  NodePhase = "Succeeded"  // the node has finished successfully
	NodePhaseResolved   NodePhase = "Resolved"   // the node's dependencies are all resolved
	NodePhaseActive     NodePhase = "Active"     // the node is active and waiting on dependencies to resolve
	NodePhaseInit       NodePhase = "Init"       // the node is initializing
	NodePhaseUnresolved NodePhase = "Unresolved" // the node is unresolved - timeout has been reached or constraints exceeded
	NodePhaseError      NodePhase = "Error"      // the node has encountered an error in processing
)

// Sensor is the definition of a sensor resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Sensor struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec          SensorSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status        SensorStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// SensorList is the list of Sensor resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SensorList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items       []Sensor `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// SensorSpec represents desired sensor state
type SensorSpec struct {
	// Signals is a list of the things that this sensor is dependent on. These are the inputs to this sensor.
	Signals []Signal `json:"signals" protobuf:"bytes,1,rep,name=signals"`

	// Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor.
	Triggers []Trigger `json:"triggers" protobuf:"bytes,2,rep,name=triggers"`

	// Escalation describes the policy for signal failures and violations of the dependency's constraints
	Escalation EscalationPolicy `json:"escalation,omitempty" protobuf:"bytes,3,opt,name=escalation"`

	// Repeat is a flag that determines if the sensor status should be reset after completion - expiremental for real-time use cases
	Repeat bool `json:"repeat,omitempty" protobuf:"bytes,4,opt,name=repeat"`
}

// Signal describes a dependency
type Signal struct {
	// Name is a unique name of this dependency
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Deadline is the duration in seconds after the StartedAt time of the sensor after which this signal is terminated
	// This trumps the recurrence patterns of calendar signal and allows a calendar signal to have a strict defined life
	// After the deadline is reached and this signal has not in a Resolved state, this signal is marked as Failed
	Deadline int64 `json:"deadline,omitempty" protobuf:"bytes,2,opt,name=deadline"`

	// NATS defines a stream based dependency
	NATS *NATS `json:"nats,omitempty" protobuf:"bytes,3,opt,name=nats"`

	// MQTT (Message Queueing Telemetry Transport) defines a pubsub-based messaging broker and topic. See ISO/IEC PRF 20922 for reference.
	MQTT *MQTT `json:"mqtt,omitempty" protobuf:"bytes,4,opt,name=mqtt"`

	// AMQP (Advanced Message Queueing Protocol) defines app layer message oriented middleware. See ISO/IEC 19464 for reference.
	AMQP *AMQP `json:"amqp,omitempty" protobuf:"bytes,5,opt,name=amqp"`

	// Kafka defines a kafka stream
	Kafka *Kafka `json:"kafka,omitempty" protobuf:"bytes,6,opt,name=kafka"`

	// artifact defines an external file dependency
	Artifact *ArtifactSignal `json:"artifact,omitempty" protobuf:"bytes,7,opt,name=artifact"`

	// Calendar defines a time based dependency
	Calendar *CalendarSignal `json:"calendar,omitempty" protobuf:"bytes,8,opt,name=calendar"`

	// Resource defines a dependency on a kubernetes resource -- this can be a pod, deployment or custom resource
	Resource *ResourceSignal `json:"resource,omitempty" protobuf:"bytes,9,opt,name=resource"`

	// Webhook defines a HTTP notification dependency
	Webhook *WebhookSignal `json:"webhook,omitempty" protobuf:"bytes,10,opt,name=webhook"`

	// Constraints and rules governing tolerations of success and overrides
	Constraints SignalConstraints `json:"constraints,omitempty" protobuf:"bytes,11,opt,name=constraints"`
}

// ArtifactSignal describes an external object dependency
type ArtifactSignal struct {
	ArtifactLocation `json:",inline" protobuf:"bytes,2,opt,name=artifactLocation"`

	// NotificationStream is the stream to listen for artifact notifications
	NotificationStream Stream `json:"stream" protobuf:"bytes,1,opt,name=stream"`
}

// CalendarSignal describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type CalendarSignal struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule" protobuf:"bytes,1,opt,name=schedule"`

	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval" protobuf:"bytes,2,opt,name=interval"`

	// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events
	// RDATE defines the list of DATE-TIME values for recurring events
	// EXDATE defines the list of DATE-TIME exceptions for recurring events
	// the combination of these rules and dates combine to form a set of date times
	Recurrence []string `json:"recurrence" protobuf:"bytes,3,rep,name=recurrence"`
}

// GroupVersionKind unambiguously identifies a kind.  It doesn't anonymously include GroupVersion
// to avoid automatic coercion.  It doesn't use a GroupVersion to avoid custom marshalling
type GroupVersionKind struct {
	Group   string `json:"group" protobuf:"bytes,1,opt,name=group"`
	Version string `json:"version" protobuf:"bytes,2,opt,name=version"`
	Kind    string `json:"kind" protobuf:"bytes,3,opt,name=kind"`
}

// ResourceSignal refers to a dependency on a k8 resource
type ResourceSignal struct {
	GroupVersionKind `json:",inline" protobuf:"bytes,3,opt,name=groupVersionKind"`
	Namespace        string          `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	Filter           *ResourceFilter `json:"filter,omitempty" protobuf:"bytes,2,opt,name=filter"`
}

// SignalConstraints defines constraints for a dependent signal
type SignalConstraints struct {
	// Time constraints on the signal
	Time TimeConstraints `json:"time,omitempty" protobuf:"bytes,1,opt,name=time"`
}

// TimeConstraints describes constraints in time
type TimeConstraints struct {
	Start v1.Time `json:"start" protobuf:"bytes,1,opt,name=start"`
	Stop  v1.Time `json:"stop" protobuf:"bytes,2,opt,name=stop"`
}

// Trigger is an action taken, output produced, an event created, a message sent
type Trigger struct {
	// Name is a unique name of the action to take
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Resource describes the resource that will be created by this action
	Resource *ResourceObject `json:"resource,omitempty" protobuf:"bytes,2,opt,name=resource"`

	// Message describes a message that will be sent on a queue
	Message *Message `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`

	// RetryStrategy is the strategy to retry a trigger if it fails
	RetryStrategy *RetryStrategy `json:"retryStrategy" protobuf:"bytes,4,opt,name=replyStrategy"`
}

// ResourceObject is the resource object to create on kubernetes
type ResourceObject struct {
	// Namespace in which to create this object
	// optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`

	// The unambiguous kind of this object - used in order to retrieve the appropriate kubernetes api client for this resource
	GroupVersionKind `json:",inline" protobuf:"bytes,4,opt,name=groupVersionKind"`

	// Location in which the K8 resource file(s) are stored.
	// If omitted, will attempt to use the default artifact location configured in the controller.
	ArtifactLocation *ArtifactLocation `json:"artifactLocation,omitempty" protobuf:"bytes,2,opt,name=artifactLocation"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. This overrides any labels in the unstructured object with the same key.
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,3,rep,name=labels"`
}

// Stream describes a queue stream resource
type Stream struct {
	NATS  *NATS  `json:"nats,omitempty" protobuf:"bytes,1,opt,name=nats"`
	MQTT  *MQTT  `json:"mqtt,omitempty" protobuf:"bytes,2,opt,name=mqtt"`
	AMQP  *AMQP  `json:"amqp,omitempty" protobuf:"bytes,3,opt,name=amqp"`
	Kafka *Kafka `json:"kafka,omitempty" protobuf:"bytes,4,opt,name=kafka"`
}

// NATS contains information to interact with a NATS messaging system
type NATS struct {
	// URL is the exposed service for client connections to a NATS cluster
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`

	// Subject is the name of the subject to subscribe to
	Subject string `json:"subject" protobuf:"bytes,2,opt,name=subject"`
}

// MQTT (Message Queuing Telemetry Transport) is an ISO standard (ISO/IEC PRF 20922)[2] publish-subscribe-based messaging protocol
type MQTT struct {
	// URL of the message broker
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`

	// Topic of interest
	Topic string `json:"topic" protobuf:"bytes,2,opt,name=topic"`
}

// AMQP (Advanced Message Queueing Protocol) defines app layer message oriented middleware. See ISO/IEC 19464 for reference.
// A RabbitMQ client is used to interface
type AMQP struct {
	URL          string `json:"url" protobuf:"bytes,1,opt,name=url"`
	ExchangeName string `json:"exchangeName" protobuf:"bytes,2,opt,name=exchangeName"`
	ExchangeType string `json:"exchangeType" protobuf:"bytes,3,opt,name=exchangeType"`
	RoutingKey   string `json:"routingKey" protobuf:"bytes,4,opt,name=routingKey"`
}

// Kafka defines a Kafka stream
type Kafka struct {
	// URL of the kafka message broker
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Topic of the kafka stream
	Topic string `json:"topic" protobuf:"bytes,2,opt,name=topic"`
	// Partition of the kafka stream
	Partition int32 `json:"partition" protobuf:"bytes,3,opt,name=partition"`
}

// WebhookSignal is a general purpose REST API
type WebhookSignal struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`
	// Port to listen on
	Port int `json:"port" protobuf:"bytes,2,opt,name=port"`
	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,3,opt,name=method"`
}

// Message represents a message on a queue
type Message struct {
	Body string `json:"body" protobuf:"bytes,1,opt,name=body"`

	// Stream descibes queue resources to send the message on
	Stream Stream `json:"stream,omitempty" protobuf:"bytes,2,opt,name=stream"`
}

// RetryStrategy represents a strategy for retrying operations
type RetryStrategy struct {
}

// EscalationPolicy describes the policy for escalating sensors in an Error state
type EscalationPolicy struct {
	// Level is the degree of importance
	Level string `json:"level" protobuf:"bytes,1,opt,name=level"`

	// need someway to progressively get more serious notifications
	Message Message `json:"message" protobuf:"bytes,2,opt,name=message"`
}

// SensorStatus contains information about the status of a sensor
type SensorStatus struct {
	// Phase is the high-level summary of the sensor
	Phase NodePhase `json:"phase" protobuf:"bytes,1,opt,name=phase"`

	// StartedAt is the time at which this sensor was initiated
	StartedAt v1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`

	// ResolvedAt is the time at which this sensor was resolved
	ResolvedAt v1.Time `json:"resolvedAt,omitempty" protobuf:"bytes,3,opt,name=resolvedAt"`

	// Message is a human readable string indicating details about a sensor in its phase
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// Nodes is a mapping between a node ID and the node's status
	// it records the states for the FSM of this sensor
	Nodes map[string]NodeStatus `json:"nodes,omitempty" protobuf:"bytes,5,rep,name=nodes"`

	// Escalated is a flag for whether this sensor was escalated
	Escalated bool `json:"escalated,omitempty" protobuf:"bytes,6,opt,name=escalated"`
}

// NodeStatus describes the status for an individual node in the sensor's FSM
type NodeStatus struct {
	// ID is a unique identifier of a node within a sensor
	// It is a hash of the node name
	ID string `json:"id" protobuf:"bytes,1,opt,name=id"`

	// Name is a unique name in the node tree used to generate the node ID
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`

	// DisplayName is the human readable representation of the node
	DisplayName string `json:"displayName" protobuf:"bytes,3,opt,name=displayName"`

	// Type is the type of the node
	Type NodeType `json:"type" protobuf:"bytes,4,opt,name=type"`

	// Phase of the node
	Phase NodePhase `json:"phase" protobuf:"bytes,5,opt,name=phase"`

	// StartedAt is the time at which this node started
	StartedAt v1.Time `json:"startedAt,omitempty" protobuf:"bytes,6,opt,name=startedAt"`

	// ResolvedAt is the time at which this node resolved
	ResolvedAt v1.Time `json:"resolvedAt,omitempty" protobuf:"bytes,7,opt,name=resolvedAt"`

	// store data or something to save for signal notifications or trigger events
	Message string `json:"message,omitempty" protobuf:"bytes,8,opt,name=message"`
}

// ArtifactLocation describes the location for an external artifact
type ArtifactLocation struct {
	S3 *S3Artifact `json:"s3,omitempty" protobuf:"bytes,1,opt,name=s3"`
}

// S3Artifact contains information about an artifact in S3
type S3Artifact struct {
	S3Bucket `json:",inline" protobuf:"bytes,5,opt,name=s3Bucket"`
	Key      string                      `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
	Event    minio.NotificationEventType `json:"event,omitempty" protobuf:"bytes,2,opt,name=event"`
	ARN      *ARN                        `json:"arn,omitempty" protobuf:"bytes,3,opt,name=arn"`
	Filter   *S3Filter                   `json:"filter,omitempty" protobuf:"bytes,4,opt,name=filter"`
}

// ARN - holds ARN information that will be sent to the web service
// ARN desciption can be found in http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
type ARN struct {
	Partition string `json:"partition" protobuf:"bytes,1,opt,name=partition"`
	Service   string `json:"service" protobuf:"bytes,2,opt,name=service"`
	Region    string `json:"region" protobuf:"bytes,3,opt,name=region"`
	AccountID string `json:"accountID" protobuf:"bytes,4,opt,name=accountID"`
	Resource  string `json:"resource" protobuf:"bytes,5,opt,name=resource"`
}

// S3Bucket contains information for an S3 Bucket
type S3Bucket struct {
	Endpoint  string                  `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`
	Bucket    string                  `json:"bucket,omitempty" protobuf:"bytes,2,opt,name=bucket"`
	Region    string                  `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
	Insecure  bool                    `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`
	AccessKey apiv1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,5,opt,name=accessKey"`
	SecretKey apiv1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,6,opt,name=secretKey"`
}

// S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects
type S3Filter struct {
	Prefix string `json:"prefix" protobuf:"bytes,1,opt,name=prefix"`
	Suffix string `json:"suffix" protobuf:"bytes,2,opt,name=suffix"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource signal objects
type ResourceFilter struct {
	Prefix      string            `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	Labels      map[string]string `json:"labels,omitempty" protobuf:"bytes,2,rep,name=labels"`
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,3,rep,name=annotations"`
	CreatedBy   v1.Time           `json:"createdBy,omitempty" protobuf:"bytes,4,opt,name=createdBy"`
}

// HasLocation whether or not an artifact has a location defined
func (a *ArtifactLocation) HasLocation() bool {
	return a.S3 != nil
}

// GetType returns the type of this signal
func (signal *Signal) GetType() SignalType {
	if signal.NATS != nil {
		return SignalTypeNats
	}
	if signal.MQTT != nil {
		return SignalTypeMQTT
	}
	if signal.AMQP != nil {
		return SignalTypeAMQP
	}
	if signal.Resource != nil {
		return SignalTypeResource
	}
	if signal.Artifact != nil {
		return SignalTypeArtifact
	}
	if signal.Calendar != nil {
		return SignalTypeCalendar
	}
	if signal.Webhook != nil {
		return SignalTypeWebhook
	}
	return "Unknown"
}

// GetType returns the type of this stream
func (stream *Stream) GetType() StreamType {
	if stream.NATS != nil {
		return StreamTypeNats
	}
	if stream.MQTT != nil {
		return StreamTypeMQTT
	}
	if stream.AMQP != nil {
		return StreamTypeAMQP
	}
	if stream.Kafka != nil {
		return StreamTypeKafka
	}
	return "Unknown"
}

// IsResolved determines if the node is resolved
func (node NodeStatus) IsResolved() bool {
	return node.Phase == NodePhaseResolved
}

// IsComplete determines if the node has reached a complete (end) state
func (node NodeStatus) IsComplete() bool {
	return node.Phase == NodePhaseSucceeded ||
		node.Phase == NodePhaseError
}

// IsResolved determines if the sensor is fully resolved for the specific nodeType
func (s *Sensor) IsResolved(nodeType NodeType) bool {
	for _, node := range s.Status.Nodes {
		if node.Type == nodeType &&
			!(node.Phase == NodePhaseResolved || node.Phase == NodePhaseSucceeded) {
			return false
		}
	}
	return true
}

// IsComplete determines if the sensor has reached a complete (end) state
func (s *Sensor) IsComplete() bool {
	if !(s.Status.Phase == NodePhaseSucceeded || s.Status.Phase == NodePhaseError) {
		return false
	}
	for _, node := range s.Status.Nodes {
		if !node.IsComplete() {
			return false
		}
	}
	return true
}

// NodeID creates a deterministic node ID based on a node name
// we support 3 kinds of "nodes" - sensors, signals, triggers
// each should pass it's name field
func (s *Sensor) NodeID(name string) string {
	if name == s.ObjectMeta.Name {
		return s.ObjectMeta.Name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return fmt.Sprintf("%s-%v", s.ObjectMeta.Name, h.Sum32())
}
