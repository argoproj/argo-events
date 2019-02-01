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
	"github.com/argoproj/argo-events/pkg/apis/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"hash/fnv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NotificationType string

const (
	EventNotification          NotificationType = "Event"
	ResourceUpdateNotification NotificationType = "ResourceUpdate"
)

// NodeType is the type of a node
type NodeType string

// possible node types
const (
	NodeTypeEventDependency NodeType = "EventDependency"
	NodeTypeTrigger         NodeType = "Trigger"
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
	v1.TypeMeta `json:",inline"`

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
	// Dependencies is a list of the events that this sensor is dependent on.
	Dependencies []EventDependency `json:"dependencies" protobuf:"bytes,1,rep,name=dependencies"`

	// Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor.
	Triggers []Trigger `json:"triggers" protobuf:"bytes,2,rep,name=triggers"`

	// DeploySpec contains sensor pod specification. For more information, read https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#pod-v1-core
	DeploySpec *corev1.PodSpec `json:"deploySpec" protobuf:"bytes,3,opt,name=deploySpec"`

	// EventProtocol is the protocol through which sensor receives events from gateway
	EventProtocol *apicommon.EventProtocol `json:"eventProtocol" protobuf:"bytes,4,opt,name=eventProtocol"`
}

// EventDependency describes a dependency
type EventDependency struct {
	// Name is a unique name of this dependency
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Deadline is the duration in seconds after the StartedAt time of the sensor after which this event is terminated.
	// Note: this functionality is not yet respected, but it's theoretical behavior is as follows:
	// This trumps the recurrence patterns of calendar events and allows any event to have a strict defined life.
	// After the deadline is reached and this event has not in a Resolved state, this event is marked as Failed
	// and proper escalations should proceed.
	Deadline int64 `json:"deadline,omitempty" protobuf:"bytes,2,opt,name=deadline"`

	// Filters and rules governing tolerations of success and constraints on the context and data of an event
	Filters EventDependencyFilter `json:"filters,omitempty" protobuf:"bytes,3,opt,name=filters"`

	// Connected tells if subscription is already setup in case of nats protocol.
	Connected bool `json:"connected,omitempty" protobuf:"bytes,4,opt,name=connected"`
}

// GroupVersionKind unambiguously identifies a kind.  It doesn't anonymously include GroupVersion
// to avoid automatic coercion.  It doesn't use a GroupVersion to avoid custom marshalling.
type GroupVersionKind struct {
	Group   string `json:"group" protobuf:"bytes,1,opt,name=group"`
	Version string `json:"version" protobuf:"bytes,2,opt,name=version"`
	Kind    string `json:"kind" protobuf:"bytes,3,opt,name=kind"`
}

// EventDependencyFilter defines filters and constraints for a event.
type EventDependencyFilter struct {
	// Name is the name of event filter
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Time filter on the event with escalation
	Time *TimeFilter `json:"time,omitempty" protobuf:"bytes,2,opt,name=time"`

	// Context filter constraints with escalation
	Context *common.EventContext `json:"context,omitempty" protobuf:"bytes,3,opt,name=context"`

	// Data filter constraints with escalation
	Data *Data `json:"data,omitempty" protobuf:"bytes,4,rep,name=data"`
}

// TimeFilter describes a window in time.
// Filters out event events that occur outside the time limits.
// In other words, only events that occur after Start and before Stop
// will pass this filter.
type TimeFilter struct {
	// Start is the beginning of a time window.
	// Before this time, events for this event are ignored and
	// format is hh:mm:ss
	Start string `json:"start,omitempty" protobuf:"bytes,1,opt,name=start"`

	// StopPattern is the end of a time window.
	// After this time, events for this event are ignored and
	// format is hh:mm:ss
	Stop string `json:"stop,omitempty" protobuf:"bytes,2,opt,name=stop"`
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
	// Src contains a source reference to the value of the resource parameter from a event event
	Src *ResourceParameterSource `json:"src" protobuf:"bytes,1,opt,name=src"`

	// Dest is the JSONPath of a resource key.
	// A path is a series of keys separated by a dot. The colon character can be escaped with '.'
	// The -1 key can be used to append a value to an existing array.
	// See https://github.com/tidwall/sjson#path-syntax for more information about how this is used.
	Dest string `json:"dest" protobuf:"bytes,2,opt,name=dest"`
}

// ResourceParameterSource defines the source for a resource parameter from a event event
type ResourceParameterSource struct {
	// Event is the name of the event for which to retrieve this event
	Event string `json:"event" protobuf:"bytes,1,opt,name=event"`

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
// A single node can represent the status for event or a trigger.
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

	// store data or something to save for event notifications or trigger events
	Message string `json:"message,omitempty" protobuf:"bytes,8,opt,name=message"`

	// Event stores the last seen event for this node
	Event *apicommon.Event `json:"event,omitempty" protobuf:"bytes,9,opt,name=event"`
}

// ArtifactLocation describes the source location for an external artifact
type ArtifactLocation struct {
	S3        *apicommon.S3Artifact `json:"s3,omitempty" protobuf:"bytes,1,opt,name=s3"`
	Inline    *string               `json:"inline,omitempty" protobuf:"bytes,2,opt,name=inline"`
	File      *FileArtifact         `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`
	URL       *URLArtifact          `json:"url,omitempty" protobuf:"bytes,4,opt,name=url"`
	Configmap *ConfigmapArtifact    `json:"configmap,omitempty" protobuf:"bytes,5,opt,name=configmap"`
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

// FileArtifact contains information about an artifact in a filesystem
type FileArtifact struct {
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
}

// URLArtifact contains information about an artifact at an http endpoint.
type URLArtifact struct {
	Path       string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
	VerifyCert bool   `json:"verifyCert,omitempty" protobuf:"bytes,2,opt,name=verifyCert"`
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
// we support 3 kinds of "nodes" - sensors, events, triggers
// each should pass it's name field
func (s *Sensor) NodeID(name string) string {
	if name == s.ObjectMeta.Name {
		return s.ObjectMeta.Name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return fmt.Sprintf("%s-%v", s.ObjectMeta.Name, h.Sum32())
}
