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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
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
	NodePhaseComplete NodePhase = "Complete" // the node has finished successfully
	NodePhaseActive   NodePhase = "Active"   // the node is active and waiting on dependencies to resolve
	NodePhaseError    NodePhase = "Error"    // the node has encountered an error in processing
	NodePhaseNew      NodePhase = ""         // the node is new
)

// Sensor is the definition of a sensor resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
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

	// Repeat is a flag that determines if the sensor status should be reset after completion.
	// NOTE: functionality is currently experimental and part of an initiative to define
	// a more concrete pattern or cycle for sensor repetition.
	Repeat bool `json:"repeat,omitempty" protobuf:"bytes,4,opt,name=repeat"`

	// ImagePullPolicy determines the when the image should be pulled from docker repository
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty" protobuf:"bytes,5,opt,name=imagePullPolicy"`

	// ServiceAccountName required for role based access
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,6,opt,name=serviceAccountName"`

	// EnvVars are user defined env variables to sensor pod
	EnvVars []corev1.EnvVar `json:"envVars,omitempty" protobuf:"bytes,7,opt,name=envVars"`

	// ImageVersion is the sensor image version to run
	ImageVersion string `json:"imageVersion,omitempty" protobuf:"bytes,8,opt,name=imageVersion"`
}

// Signal describes a dependency
type Signal struct {
	// Name is a unique name of this dependency
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Deadline is the duration in seconds after the StartedAt time of the sensor after which this signal is terminated.
	// Note: this functionality is not yet respected, but it's theoretical behavior is as follows:
	// This trumps the recurrence patterns of calendar signals and allows any signal to have a strict defined life.
	// After the deadline is reached and this signal has not in a Resolved state, this signal is marked as Failed
	// and proper escalations should proceed.
	Deadline int64 `json:"deadline,omitempty" protobuf:"bytes,2,opt,name=deadline"`

	// Filters and rules governing tolerations of success and constraints on the context and data of an event
	Filters SignalFilter `json:"filters,omitempty" protobuf:"bytes,3,opt,name=filters"`
}

// GroupVersionKind unambiguously identifies a kind.  It doesn't anonymously include GroupVersion
// to avoid automatic coercion.  It doesn't use a GroupVersion to avoid custom marshalling.
type GroupVersionKind struct {
	Group   string `json:"group" protobuf:"bytes,1,opt,name=group"`
	Version string `json:"version" protobuf:"bytes,2,opt,name=version"`
	Kind    string `json:"kind" protobuf:"bytes,3,opt,name=kind"`
}

// SignalFilter defines filters and constraints for a signal.
type SignalFilter struct {
	// Name is the name of signal filter
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Time filter on the signal with escalation
	Time *TimeFilter `json:"time,omitempty" protobuf:"bytes,2,opt,name=time"`

	// Context filter constraints with escalation
	Context *EventContext `json:"context,omitempty" protobuf:"bytes,3,opt,name=context"`

	// Data filter constraints with escalation
	Data *Data `json:"data,omitempty" protobuf:"bytes,4,rep,name=data"`
}

// TimeFilter describes a window in time.
// Filters out signal events that occur outside the time limits.
// In other words, only events that occur after Start and before Stop
// will pass this filter.
type TimeFilter struct {
	// Start is the beginning of a time window.
	// Before this time, events for this signal are ignored and
	// format is hh:mm:ss
	Start string `json:"start,omitempty" protobuf:"bytes,1,opt,name=start"`

	// StopPattern is the end of a time window.
	// After this time, events for this signal are ignored and
	// format is hh:mm:ss
	Stop string `json:"stop,omitempty" protobuf:"bytes,2,opt,name=stop"`

	// EscalationPolicy is the escalation to trigger in case the signal filter fails
	EscalationPolicy *EscalationPolicy `json:"escalationPolicy,omitempty" protobuf:"bytes,3,opt,name=escalationPolicy"`
}

// JSONType contains the supported JSON types for data filtering
type JSONType string

// the various supported JSONTypes
const (
	JSONTypeBool   JSONType = "bool"
	JSONTypeNumber JSONType = "number"
	JSONTypeString JSONType = "string"
)

type Data struct {
	// filter constraints
	Filters []*DataFilter `json:"filters" protobuf:"bytes,1,rep,name=filters"`

	// EscalationPolicy is the escalation to trigger in case the signal filter fails
	EscalationPolicy *EscalationPolicy `json:"escalationPolicy,omitempty" protobuf:"bytes,2,opt,name=escalationPolicy"`
}

// DataFilter describes constraints and filters for event data
// Regular Expressions are purposefully not a feature as they are overkill for our uses here
// See Rob Pike's Post: https://commandcenter.blogspot.com/2011/08/regular-expressions-in-lexing-and.html
type DataFilter struct {
	// Path is the JSONPath of the event's (JSON decoded) data key
	// Path is a series of keys separated by a dot. A key may contain wildcard characters '*' and '?'.
	// To access an array value use the index as the key. The dot and wildcard characters can be escaped with '\\'.
	// See https://github.com/tidwall/gjson#path-syntax for more information on how to use this.
	Path string `json:"path" protobuf:"bytes,1,opt,name=path"`

	// Type contains the JSON type of the data
	Type JSONType `json:"type" protobuf:"bytes,2,opt,name=type"`

	// Value is the expected string value for this key
	// Booleans are pased using strconv.ParseBool()
	// Numbers are parsed using as float64 using strconv.ParseFloat()
	// Strings are taken as is
	// Nils this value is ignored
	Value string `json:"value" protobuf:"bytes,3,opt,name=value"`

	// EscalationPolicy is the escalation to trigger in case the signal filter fails
	EscalationPolicy *EscalationPolicy `json:"escalationPolicy,omitempty" protobuf:"bytes,4,opt,name=escalationPolicy"`
}

// Trigger is an action taken, output produced, an event created, a message sent
type Trigger struct {
	// Name is a unique name of the action to take
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Resource describes the resource that will be created by this action
	Resource *ResourceObject `json:"resource,omitempty" protobuf:"bytes,2,opt,name=resource"`

	// Message describes a message that will be sent on a queue
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`

	// RetryStrategy is the strategy to retry a trigger if it fails
	RetryStrategy *RetryStrategy `json:"retryStrategy" protobuf:"bytes,4,opt,name=replyStrategy"`
}

// ResourceParameter indicates a passed parameter to a service template
type ResourceParameter struct {
	// Src contains a source reference to the value of the resource parameter from a signal event
	Src *ResourceParameterSource `json:"src" protobuf:"bytes,1,opt,name=src"`

	// Dest is the JSONPath of a resource key.
	// A path is a series of keys separated by a dot. The colon character can be escaped with '.'
	// The -1 key can be used to append a value to an existing array.
	// See https://github.com/tidwall/sjson#path-syntax for more information about how this is used.
	Dest string `json:"dest" protobuf:"bytes,2,opt,name=dest"`
}

// ResourceParameterSource defines the source for a resource parameter from a signal event
type ResourceParameterSource struct {
	// Signal is the name of the signal for which to retrieve this event
	Signal string `json:"signal" protobuf:"bytes,1,opt,name=signal"`

	// Path is the JSONPath of the event's (JSON decoded) data key
	// Path is a series of keys separated by a dot. A key may contain wildcard characters '*' and '?'.
	// To access an array value use the index as the key. The dot and wildcard characters can be escaped with '\\'.
	// See https://github.com/tidwall/gjson#path-syntax for more information on how to use this.
	Path string `json:"path" protobuf:"bytes,2,opt,name=path"`

	// Value is the default literal value to use for this parameter source
	// This is only used if the path is invalid.
	// If the path is invalid and this is not defined, this param source will produce an error.
	Value *string `json:"value,omitempty" protobuf:"bytes,3,opt,name=value"`
}

// ResourceObject is the resource object to create on kubernetes
type ResourceObject struct {
	// The unambiguous kind of this object - used in order to retrieve the appropriate kubernetes api client for this resource
	GroupVersionKind `json:",inline" protobuf:"bytes,5,opt,name=groupVersionKind"`

	// Namespace in which to create this object
	// optional
	// defaults to the service account namespace
	Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`

	// Source of the K8 resource file(s)
	Source ArtifactLocation `json:"source" protobuf:"bytes,6,opt,name=source"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. This overrides any labels in the unstructured object with the same key.
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,3,rep,name=labels"`

	// Parameters is the list of resource parameters to pass in the object
	Parameters []ResourceParameter `json:"parameters" protobuf:"bytes,4,rep,name=parameters"`
}

// RetryStrategy represents a strategy for retrying operations
// TODO: implement me
type RetryStrategy struct {
}

// EscalationLevel is the degree of importance
type EscalationLevel string

// possible values for EscalationLevel
const (
	// level 0 of escalation
	Alert EscalationLevel = "Alert"
	// level 1 of escalation
	Warning EscalationLevel = "Warning"
	// level 2 of escalation
	Critical EscalationLevel = "Critical"
)

// EscalationPolicy describes the policy for escalating sensors in an Error state.
// An escalation policy is associated with signal filter. Whenever a signal filter fails
// escalation will be triggered
type EscalationPolicy struct {
	// Name is name of the escalation policy
	// This is referred by signal filter/s
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Level is the degree of importance
	Level EscalationLevel `json:"level" protobuf:"bytes,2,opt,name=level"`

	// need someway to progressively get more serious notifications
	Message string `json:"message" protobuf:"bytes,3,opt,name=message"`
}

// SensorStatus contains information about the status of a sensor.
type SensorStatus struct {
	// Phase is the high-level summary of the sensor
	Phase NodePhase `json:"phase" protobuf:"bytes,1,opt,name=phase"`

	// StartedAt is the time at which this sensor was initiated
	StartedAt v1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`

	// CompletedAt is the time at which this sensor was completed
	CompletedAt v1.Time `json:"completedAt,omitempty" protobuf:"bytes,3,opt,name=completedAt"`

	// CompletionCount is the count of sensor's successful runs.
	CompletionCount int32 `json:"completionCount,omitempty" protobuf:"varint,6,opt,name=completionCount"`

	// Message is a human readable string indicating details about a sensor in its phase
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// Nodes is a mapping between a node ID and the node's status
	// it records the states for the FSM of this sensor.
	Nodes map[string]NodeStatus `json:"nodes,omitempty" protobuf:"bytes,5,rep,name=nodes"`
}

// NodeStatus describes the status for an individual node in the sensor's FSM.
// A single node can represent the status for signal or a trigger.
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
	StartedAt v1.MicroTime `json:"startedAt,omitempty" protobuf:"bytes,6,opt,name=startedAt"`

	// CompletedAt is the time at which this node completed
	CompletedAt v1.MicroTime `json:"completedAt,omitempty" protobuf:"bytes,7,opt,name=completedAt"`

	// store data or something to save for signal notifications or trigger events
	Message string `json:"message,omitempty" protobuf:"bytes,8,opt,name=message"`

	// LatestEvent stores the last seen event for this node
	LatestEvent *EventWrapper `json:"latestEvent,omitempty" protobuf:"bytes,9,opt,name=latestEvent"`
}

// EventWrapper wraps an event with an additional flag to check if we processed this event already
type EventWrapper struct {
	Event `json:"event" protobuf:"bytes,1,opt,name=event"`
	Seen  bool `json:"seen" protobuf:"bytes,2,opt,name=seen"`
}

// Event is a data record expressing an occurrence and its context.
// Adheres to the CloudEvents v0.1 specification
type Event struct {
	Context EventContext `json:"context" protobuf:"bytes,1,opt,name=context"`
	Payload []byte       `json:"payload" protobuf:"bytes,2,opt,name=data"`
}

// EventContext contains metadata that provides circumstantial information about the occurrence.
type EventContext struct {
	// The type of occurrence which has happened. Often this attribute is used for
	// routing, observability, policy enforcement, etc.
	// should be prefixed with a reverse-DNS name. The prefixed domain dictates
	// the organization which defines the semantics of this event type. ex: com.github.pull.create
	EventType string `json:"eventType" protobuf:"bytes,1,opt,name=eventType"`

	// The version of the eventType. Enables the interpretation of data by eventual consumers,
	// requires the consumer to be knowledgeable about the producer.
	EventTypeVersion string `json:"eventTypeVersion" protobuf:"bytes,2,opt,name=eventTypeVersion"`

	// The version of the CloudEvents specification which the event uses.
	// Enables the interpretation of the context.
	CloudEventsVersion string `json:"cloudEventsVersion" protobuf:"bytes,3,opt,name=cloudEventsVersion"`

	// This describes the event producer.
	Source *URI `json:"source" protobuf:"bytes,4,opt,name=source"`

	// ID of the event. The semantics are explicitly undefined to ease the implementation of producers.
	// Enables deduplication. Must be unique within scope of producer.
	EventID string `json:"eventID" protobuf:"bytes,5,opt,name=eventID"`

	// Timestamp of when the event happened. Must adhere to format specified in RFC 3339.
	EventTime v1.MicroTime `json:"eventTime" protobuf:"bytes,6,opt,name=eventTime"`

	// A link to the schema that the data attribute adheres to.
	// Must adhere to the format specified in RFC 3986.
	SchemaURL *URI `json:"schemaURL" protobuf:"bytes,7,opt,name=schemaURL"`

	// Content type of the data attribute value. Enables the data attribute to carry any type of content,
	// whereby format and encoding might differ from that of the chosen event format.
	// For example, the data attribute may carry an XML or JSON payload and the consumer is informed
	// by this attribute being set to "application/xml" or "application/json" respectively.
	ContentType string `json:"contentType" protobuf:"bytes,8,opt,name=contentType"`

	// This is for additional metadata and does not have a mandated structure.
	// Enables a place for custom fields a producer or middleware might want to include and provides a place
	// to test metadata before adding them to the CloudEvents specification.
	Extensions map[string]string `json:"extensions,omitempty" protobuf:"bytes,9,rep,name=extensions"`

	// EscalationPolicy is the name of escalation policy to trigger in case the signal filter fails
	EscalationPolicy *EscalationPolicy `json:"escalationPolicy,omitempty" protobuf:"bytes,10,opt,name=escalationPolicy"`
}

// URI is a Uniform Resource Identifier based on RFC 3986
type URI struct {
	Scheme   string `json:"scheme" protobuf:"bytes,1,opt,name=scheme"`
	User     string `json:"user" protobuf:"bytes,2,opt,name=user"`
	Password string `json:"password" protobuf:"bytes,3,opt,name=password"`
	Host     string `json:"host" protobuf:"bytes,4,opt,name=host"`
	Port     int32  `json:"port" protobuf:"bytes,5,opt,name=port"`
	Path     string `json:"path" protobuf:"bytes,6,opt,name=path"`
	Query    string `json:"query" protobuf:"bytes,7,opt,name=query"`
	Fragment string `json:"fragment" protobuf:"bytes,8,opt,name=fragment"`
}

// ArtifactLocation describes the source location for an external artifact
type ArtifactLocation struct {
	S3        *S3Artifact        `json:"s3,omitempty" protobuf:"bytes,1,opt,name=s3"`
	Inline    *string            `json:"inline,omitempty" protobuf:"bytes,2,opt,name=inline"`
	File      *FileArtifact      `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`
	URL       *URLArtifact       `json:"url,omitempty" protobuf:"bytes,4,opt,name=url"`
	Configmap *ConfigmapArtifact `json:"configmap,omitempty" protobuf:"bytes,5,opt,name=configmap"`
}

// ConfigmapArtifact contains information about artifact in k8 configmap
type ConfigmapArtifact struct {
	// Name of the configmap
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Namespace where configmap is deployed
	Namespace string `json:"namespace" protobuf:"bytes,2,opt,name=namespace"`
	// Key within configmap data which contains trigger resource definition
	Key string `json:"key" protobuf:"bytes,3,opt,name=key"`
}

// S3Artifact contains information about an artifact in S3
type S3Artifact struct {
	S3Bucket `json:",inline" protobuf:"bytes,4,opt,name=s3Bucket"`
	Key      string                      `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
	Event    minio.NotificationEventType `json:"event,omitempty" protobuf:"bytes,2,opt,name=event"`
	Filter   *S3Filter                   `json:"filter,omitempty" protobuf:"bytes,3,opt,name=filter"`
}

// FileArtifact contains information about an artifact in a filesystem
type FileArtifact struct {
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
}

// URLArtifact contains information about an artifact at an http endpoint.
type URLArtifact struct {
	Path       string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
	VerifyCert bool   `json:"verifyCert,omitempty" protobuf:"bytes,2,opt,name=verifyCert"`
}

// S3Bucket contains information for an S3 Bucket
type S3Bucket struct {
	Endpoint  string                   `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`
	Bucket    string                   `json:"bucket,omitempty" protobuf:"bytes,2,opt,name=bucket"`
	Region    string                   `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
	Insecure  bool                     `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`
	AccessKey corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,5,opt,name=accessKey"`
	SecretKey corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,6,opt,name=secretKey"`
}

// S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects
type S3Filter struct {
	Prefix string `json:"prefix" protobuf:"bytes,1,opt,name=prefix"`
	Suffix string `json:"suffix" protobuf:"bytes,2,opt,name=suffix"`
}

// HasLocation whether or not an artifact has a location defined
func (a *ArtifactLocation) HasLocation() bool {
	return a.S3 != nil || a.Inline != nil || a.File != nil || a.URL != nil
}

// IsComplete determines if the node has reached an end state
func (node NodeStatus) IsComplete() bool {
	return node.Phase == NodePhaseComplete ||
		node.Phase == NodePhaseError
}

// IsComplete determines if the sensor has reached an end state
func (s *Sensor) IsComplete() bool {
	if !(s.Status.Phase == NodePhaseComplete || s.Status.Phase == NodePhaseError) {
		return false
	}
	for _, node := range s.Status.Nodes {
		if !node.IsComplete() {
			return false
		}
	}
	return true
}

// AreAllNodesSuccess determines if all nodes of the given type have completed successfully
func (s *Sensor) AreAllNodesSuccess(nodeType NodeType) bool {
	for _, node := range s.Status.Nodes {
		if node.Type == nodeType && node.Phase != NodePhaseComplete {
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
