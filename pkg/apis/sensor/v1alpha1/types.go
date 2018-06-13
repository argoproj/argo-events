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
	SignalTypeStream   SignalType = "Stream"
	SignalTypeArtifact SignalType = "Artifact"
	SignalTypeCalendar SignalType = "Calendar"
	SignalTypeResource SignalType = "Resource"
	SignalTypeWebhook  SignalType = "Webhook"
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
	NodePhaseNew        NodePhase = ""           // the node is new
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

	// Escalation describes the policy for signal failures and violations of the dependency's constraints.
	Escalation EscalationPolicy `json:"escalation,omitempty" protobuf:"bytes,3,opt,name=escalation"`

	// Repeat is a flag that determines if the sensor status should be reset after completion.
	// NOTE: functionality is currently expiremental and part of an initiative to define
	// a more concrete pattern or cycle for sensor reptition.
	Repeat bool `json:"repeat,omitempty" protobuf:"bytes,4,opt,name=repeat"`
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

	// Stream defines a message stream dependency
	Stream *Stream `json:"stream,omitempty" protobuf:"bytes,3,opt,name=stream"`

	// artifact defines an external file dependency
	Artifact *ArtifactSignal `json:"artifact,omitempty" protobuf:"bytes,4,opt,name=artifact"`

	// Calendar defines a time based dependency
	Calendar *CalendarSignal `json:"calendar,omitempty" protobuf:"bytes,5,opt,name=calendar"`

	// Resource defines a dependency on a kubernetes resource -- this can be a pod, deployment or custom resource
	Resource *ResourceSignal `json:"resource,omitempty" protobuf:"bytes,6,opt,name=resource"`

	// Webhook defines a HTTP notification dependency
	Webhook *WebhookSignal `json:"webhook,omitempty" protobuf:"bytes,7,opt,name=webhook"`

	// Constraints and rules governing tolerations of success and overrides
	Constraints SignalConstraints `json:"constraints,omitempty" protobuf:"bytes,8,opt,name=constraints"`
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
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
	// RDATE defines the list of DATE-TIME values for recurring events.
	// EXDATE defines the list of DATE-TIME exceptions for recurring events.
	// the combination of these rules and dates combine to form a set of date times.
	// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
	Recurrence []string `json:"recurrence" protobuf:"bytes,3,rep,name=recurrence"`
}

// GroupVersionKind unambiguously identifies a kind.  It doesn't anonymously include GroupVersion
// to avoid automatic coercion.  It doesn't use a GroupVersion to avoid custom marshalling.
type GroupVersionKind struct {
	Group   string `json:"group" protobuf:"bytes,1,opt,name=group"`
	Version string `json:"version" protobuf:"bytes,2,opt,name=version"`
	Kind    string `json:"kind" protobuf:"bytes,3,opt,name=kind"`
}

// ResourceSignal refers to a dependency on a k8s resource.
type ResourceSignal struct {
	GroupVersionKind `json:",inline" protobuf:"bytes,3,opt,name=groupVersionKind"`
	Namespace        string          `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	Filter           *ResourceFilter `json:"filter,omitempty" protobuf:"bytes,2,opt,name=filter"`
}

// SignalConstraints defines constraints for a dependent signal.
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
	// Type of the stream resource
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`

	// URL is the exposed endpoint for client connections to this service
	URL string `json:"url" protobuf:"bytes,2,opt,name=url"`

	// Attributes contains additional fields specific to each service implementation
	Attributes map[string]string `json:"attributes,omitempty" protobuf:"bytes,3,rep,name=attributes"`
}

// WebhookSignal is a general purpose REST API
type WebhookSignal struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`

	// Port to listen on
	Port int32 `json:"port" protobuf:"bytes,2,opt,name=port"`

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
// TODO: implement me
type RetryStrategy struct {
}

// EscalationPolicy describes the policy for escalating sensors in an Error state.
// NOTE: this functionality is currently experimental, but we believe serves as an
// important future enhancement around handling lifecycle error conditions of a sensor.
type EscalationPolicy struct {
	// Level is the degree of importance
	Level string `json:"level" protobuf:"bytes,1,opt,name=level"`

	// need someway to progressively get more serious notifications
	Message Message `json:"message" protobuf:"bytes,2,opt,name=message"`
}

// SensorStatus contains information about the status of a sensor.
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
	// it records the states for the FSM of this sensor.
	Nodes map[string]NodeStatus `json:"nodes,omitempty" protobuf:"bytes,5,rep,name=nodes"`

	// Escalated is a flag for whether this sensor was escalated
	Escalated bool `json:"escalated,omitempty" protobuf:"bytes,6,opt,name=escalated"`
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
	Filter   *S3Filter                   `json:"filter,omitempty" protobuf:"bytes,3,opt,name=filter"`
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
	if signal.Stream != nil {
		return SignalTypeStream
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
