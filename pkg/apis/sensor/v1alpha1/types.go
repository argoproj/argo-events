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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ArgoEventsSensorVersion = "v0.10"

// NotificationType represent a type of notifications that are handled by a sensor
type NotificationType string

const (
	// EventNotification is a notification for an event dependency (from a Gateway)
	EventNotification NotificationType = "Event"
	// ResourceUpdateNotification is a notification that an associated resource was updated
	ResourceUpdateNotification NotificationType = "ResourceUpdate"
)

// NodeType is the type of a node
type NodeType string

const (
	// NodeTypeEventDependency is a node that represents a single event dependency
	NodeTypeEventDependency NodeType = "EventDependency"
	// NodeTypeTrigger is a node that represents a single trigger
	NodeTypeTrigger NodeType = "Trigger"
	// NodeTypeDependencyGroup is a node that represents a group of event dependencies
	NodeTypeDependencyGroup NodeType = "DependencyGroup"
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

// TriggerCycleState is the label for the state of the trigger cycle
type TriggerCycleState string

// possible values of trigger cycle states
const (
	TriggerCycleSuccess TriggerCycleState = "Success" // all triggers are successfully executed
	TriggerCycleFailure TriggerCycleState = "Failure" // one or more triggers failed
)

// Sensor is the definition of a sensor resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type Sensor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              SensorSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            SensorStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// SensorList is the list of Sensor resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SensorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Sensor `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// SensorSpec represents desired sensor state
type SensorSpec struct {
	// Dependencies is a list of the events that this sensor is dependent on.
	Dependencies []EventDependency `json:"dependencies" protobuf:"bytes,1,rep,name=dependencies"`

	// Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor.
	Triggers []Trigger `json:"triggers" protobuf:"bytes,2,rep,name=triggers"`

	// Template contains sensor pod specification. For more information, read https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#pod-v1-core
	Template *corev1.PodTemplateSpec `json:"template" protobuf:"bytes,3,name=template"`

	// EventProtocol is the protocol through which sensor receives events from gateway
	EventProtocol *apicommon.EventProtocol `json:"eventProtocol" protobuf:"bytes,4,name=eventProtocol"`

	// Circuit is a boolean expression of dependency groups
	Circuit string `json:"circuit,omitempty" protobuf:"bytes,5,rep,name=circuit"`

	// DependencyGroups is a list of the groups of events.
	DependencyGroups []DependencyGroup `json:"dependencyGroups,omitempty" protobuf:"bytes,6,rep,name=dependencyGroups"`

	// ErrorOnFailedRound if set to true, marks sensor state as `error` if the previous trigger round fails.
	// Once sensor state is set to `error`, no further triggers will be processed.
	ErrorOnFailedRound bool `json:"errorOnFailedRound,omitempty" protobuf:"bytes,7,opt,name=errorOnFailedRound"`
}

// EventDependency describes a dependency
type EventDependency struct {
	// Name is a unique name of this dependency
	Name string `json:"name" protobuf:"bytes,1,name=name"`

	// Filters and rules governing tolerations of success and constraints on the context and data of an event
	Filters EventDependencyFilter `json:"filters,omitempty" protobuf:"bytes,2,opt,name=filters"`

	// Connected tells if subscription is already setup in case of nats protocol.
	Connected bool `json:"connected,omitempty" protobuf:"bytes,3,opt,name=connected"`
}

// DependencyGroup is the group of dependencies
type DependencyGroup struct {
	// Name of the group
	Name string `json:"name" protobuf:"bytes,1,name=name"`
	// Dependencies of events
	Dependencies []string `json:"dependencies" protobuf:"bytes,2,name=dependencies"`
}

// EventDependencyFilter defines filters and constraints for a event.
type EventDependencyFilter struct {
	// Name is the name of event filter
	Name string `json:"name" protobuf:"bytes,1,name=name"`

	// Time filter on the event with escalation
	Time *TimeFilter `json:"time,omitempty" protobuf:"bytes,2,opt,name=time"`

	// Context filter constraints with escalation
	Context *common.EventContext `json:"context,omitempty" protobuf:"bytes,3,opt,name=context"`

	// Data filter constraints with escalation
	Data []DataFilter `json:"data,omitempty" protobuf:"bytes,4,opt,name=data"`
}

// TimeFilter describes a window in time.
// DataFilters out event events that occur outside the time limits.
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

	// Value is the allowed string values for this key
	// Booleans are passed using strconv.ParseBool()
	// Numbers are parsed using as float64 using strconv.ParseFloat()
	// Strings are taken as is
	// Nils this value is ignored
	Value []string `json:"value" protobuf:"bytes,3,rep,name=value"`
}

// Trigger is an action taken, output produced, an event created, a message sent
type Trigger struct {
	// Template describes the trigger specification.
	Template *TriggerTemplate `json:"template" protobuf:"bytes,1,name=template"`

	// TemplateParameters is the list of resource parameters to pass to the template object
	TemplateParameters []TriggerParameter `json:"templateParameters,omitempty" protobuf:"bytes,2,rep,name=templateParameters"`

	// ResourceParameters is the list of resource parameters to pass to resolved resource object in template object
	ResourceParameters []TriggerParameter `json:"resourceParameters,omitempty" protobuf:"bytes,3,rep,name=resourceParameters"`

	// Policy to configure backoff and execution criteria for the trigger
	Policy *TriggerPolicy `json:"policy" protobuf:"bytes,4,opt,name=policy"`
}

// TriggerTemplate is the template that describes trigger specification.
type TriggerTemplate struct {
	// Name is a unique name of the action to take
	Name string `json:"name" protobuf:"bytes,1,name=name"`

	// When is the condition to execute the trigger
	When *TriggerCondition `json:"when,omitempty" protobuf:"bytes,2,opt,name=when"`

	// The unambiguous kind of this object - used in order to retrieve the appropriate kubernetes api client for this resource
	*metav1.GroupVersionKind `json:",inline" protobuf:"bytes,3,opt,name=groupVersionKind"`

	// Source of the K8 resource file(s)
	Source *ArtifactLocation `json:"source" protobuf:"bytes,4,opt,name=source"`
}

// TriggerCondition describes condition which must be satisfied in order to execute a trigger.
// Depending upon condition type, status of dependency groups is used to evaluate the result.
type TriggerCondition struct {
	// Any acts as a OR operator between dependencies
	Any []string `json:"any,omitempty" protobuf:"bytes,1,rep,name=any"`

	// All acts as a AND operator between dependencies
	All []string `json:"all,omitempty" protobuf:"bytes,2,rep,name=all"`
}

// TriggerParameterOperation represents how to set a trigger destination
// resource key
type TriggerParameterOperation string

const (
	// TriggerParameterOpNone is the zero value of TriggerParameterOperation
	TriggerParameterOpNone TriggerParameterOperation = ""
	// TriggerParameterOpAppend means append the new value to the existing
	TriggerParameterOpAppend TriggerParameterOperation = "append"
	// TriggerParameterOpOverwrite means overwrite the existing value with the new
	TriggerParameterOpOverwrite TriggerParameterOperation = "overwrite"
	// TriggerParameterOpPrepend means prepend the new value to the existing
	TriggerParameterOpPrepend TriggerParameterOperation = "prepend"
)

// TriggerParameter indicates a passed parameter to a service template
type TriggerParameter struct {
	// Src contains a source reference to the value of the parameter from a event event
	Src *TriggerParameterSource `json:"src" protobuf:"bytes,1,name=src"`

	// Dest is the JSONPath of a resource key.
	// A path is a series of keys separated by a dot. The colon character can be escaped with '.'
	// The -1 key can be used to append a value to an existing array.
	// See https://github.com/tidwall/sjson#path-syntax for more information about how this is used.
	Dest string `json:"dest" protobuf:"bytes,2,name=dest"`

	// Operation is what to do with the existing value at Dest, whether to
	// 'prepend', 'overwrite', or 'append' it.
	Operation TriggerParameterOperation `json:"operation,omitempty" protobuf:"bytes,3,opt,name=operation"`
}

// TriggerParameterSource defines the source for a parameter from a event event
type TriggerParameterSource struct {
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

// TriggerPolicy dictates the policy for the trigger retries
type TriggerPolicy struct {
	// Backoff before checking resource state
	Backoff Backoff `json:"backoff" protobuf:"bytes,1,opt,name=backoff"`

	// State refers to labels used to check the resource state
	State *TriggerStateLabels `json:"state" protobuf:"bytes,2,opt,name=state"`

	// ErrorOnBackoffTimeout determines whether sensor should transition to error state if the backoff times out and yet the resource neither transitioned into success or failure.
	ErrorOnBackoffTimeout bool `json:"errorOnBackoffTimeout" protobuf:"bytes,3,opt,name=errorOnBackoffTimeout"`
}

// Backoff for an operation
type Backoff struct {
	// Duration is the duration in nanoseconds
	Duration time.Duration `json:"duration" protobuf:"bytes,1,opt,name=duration"`

	// Duration is multiplied by factor each iteration
	Factor float64 `json:"factor" protobuf:"bytes,2,opt,name=factor"`

	// The amount of jitter applied each iteration
	Jitter float64 `json:"jitter" protobuf:"bytes,3,opt,name=jitter"`

	// Exit with error after this many steps
	Steps int `json:"steps" protobuf:"bytes,4,opt,name=steps"`
}

// TriggerStateLabels defines the labels used to decide if a resource is in success or failure state.
type TriggerStateLabels struct {
	// Success defines labels required to identify a resource in success state
	Success map[string]string `json:"success" protobuf:"bytes,1,opt,name=success"`

	// Failure defines labels required to identify a resource in failed state
	Failure map[string]string `json:"failure" protobuf:"bytes,2,opt,name=failure"`
}

// SensorStatus contains information about the status of a sensor.
type SensorStatus struct {
	// Phase is the high-level summary of the sensor
	Phase NodePhase `json:"phase" protobuf:"bytes,1,opt,name=phase"`

	// StartedAt is the time at which this sensor was initiated
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`

	// CompletedAt is the time at which this sensor was completed
	CompletedAt metav1.Time `json:"completedAt,omitempty" protobuf:"bytes,3,opt,name=completedAt"`

	// Message is a human readable string indicating details about a sensor in its phase
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// Nodes is a mapping between a node ID and the node's status
	// it records the states for the FSM of this sensor.
	Nodes map[string]NodeStatus `json:"nodes,omitempty" protobuf:"bytes,5,rep,name=nodes"`

	// TriggerCycleCount is the count of sensor's trigger cycle runs.
	TriggerCycleCount int32 `json:"triggerCycleCount,omitempty" protobuf:"varint,6,opt,name=triggerCycleCount"`

	// TriggerCycleState is the status from last cycle of triggers execution.
	TriggerCycleStatus TriggerCycleState `json:"triggerCycleStatus" protobuf:"bytes,7,opt,name=triggerCycleStatus"`

	// LastCycleTime is the time when last trigger cycle completed
	LastCycleTime metav1.Time `json:"lastCycleTime" protobuf:"bytes,8,opt,name=lastCycleTime"`
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
	StartedAt metav1.MicroTime `json:"startedAt,omitempty" protobuf:"bytes,6,opt,name=startedAt"`

	// CompletedAt is the time at which this node completed
	CompletedAt metav1.MicroTime `json:"completedAt,omitempty" protobuf:"bytes,7,opt,name=completedAt"`

	// store data or something to save for event notifications or trigger events
	Message string `json:"message,omitempty" protobuf:"bytes,8,opt,name=message"`

	// Event stores the last seen event for this node
	Event *apicommon.Event `json:"event,omitempty" protobuf:"bytes,9,opt,name=event"`
}

// ArtifactLocation describes the source location for an external artifact
type ArtifactLocation struct {
	// S3 compliant artifact
	S3 *apicommon.S3Artifact `json:"s3,omitempty" protobuf:"bytes,1,opt,name=s3"`

	// Inline artifact is embedded in sensor spec as a string
	Inline *string `json:"inline,omitempty" protobuf:"bytes,2,opt,name=inline"`

	// File artifact is artifact stored in a file
	File *FileArtifact `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`

	// URL to fetch the artifact from
	URL *URLArtifact `json:"url,omitempty" protobuf:"bytes,4,opt,name=url"`

	// Configmap that stores the artifact
	Configmap *ConfigmapArtifact `json:"configmap,omitempty" protobuf:"bytes,5,opt,name=configmap"`

	// Git repository hosting the artifact
	Git *GitArtifact `json:"git,omitempty" protobuf:"bytes,6,opt,name=git"`

	// Resource is generic template for K8s resource
	Resource *unstructured.Unstructured `json:"resource,omitempty" protobuf:"bytes,7,opt,name=resource"`
}

// ConfigmapArtifact contains information about artifact in k8 configmap
type ConfigmapArtifact struct {
	// Name of the configmap
	Name string `json:"name" protobuf:"bytes,1,name=name"`

	// Namespace where configmap is deployed
	Namespace string `json:"namespace" protobuf:"bytes,2,name=namespace"`

	// Key within configmap data which contains trigger resource definition
	Key string `json:"key" protobuf:"bytes,3,name=key"`
}

// FileArtifact contains information about an artifact in a filesystem
type FileArtifact struct {
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
}

// URLArtifact contains information about an artifact at an http endpoint.
type URLArtifact struct {
	// Path is the complete URL
	Path string `json:"path" protobuf:"bytes,1,name=path"`

	// VerifyCert decides whether the connection is secure or not
	VerifyCert bool `json:"verifyCert,omitempty" protobuf:"bytes,2,opt,name=verifyCert"`
}

// GitArtifact contains information about an artifact stored in git
type GitArtifact struct {
	// Git URL
	URL string `json:"url" protobuf:"bytes,1,name=url"`

	// Directory to clone the repository. We clone complete directory because GitArtifact is not limited to any specific Git service providers.
	// Hence we don't use any specific git provider client.
	CloneDirectory string `json:"cloneDirectory" protobuf:"bytes,2,name=cloneDirectory"`

	// Creds contain reference to git username and password
	// +optional
	Creds *GitCreds `json:"creds,omitempty" protobuf:"bytes,3,opt,name=creds"`

	// Namespace where creds are stored.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`

	// SSHKeyPath is path to your ssh key path. Use this if you don't want to provide username and password.
	// ssh key path must be mounted in sensor pod.
	// +optional
	SSHKeyPath string `json:"sshKeyPath,omitempty" protobuf:"bytes,5,opt,name=sshKeyPath"`

	// Path to file that contains trigger resource definition
	FilePath string `json:"filePath" protobuf:"bytes,6,name=filePath"`

	// Branch to use to pull trigger resource
	// +optional
	Branch string `json:"branch,omitempty" protobuf:"bytes,7,opt,name=branch"`

	// Tag to use to pull trigger resource
	// +optional
	Tag string `json:"tag,omitempty" protobuf:"bytes,8,opt,name=tag"`

	// Remote to manage set of tracked repositories. Defaults to "origin".
	// Refer https://git-scm.com/docs/git-remote
	// +optional
	Remote *GitRemoteConfig `json:"remote" protobuf:"bytes,9,opt,name=remote"`
}

// GitRemoteConfig contains the configuration of a Git remote
type GitRemoteConfig struct {
	// Name of the remote to fetch from.
	Name string `json:"name" protobuf:"bytes,1,name=name"`

	// URLs the URLs of a remote repository. It must be non-empty. Fetch will
	// always use the first URL, while push will use all of them.
	URLS []string `json:"urls" protobuf:"bytes,2,rep,name=urls"`
}

// GitCreds contain reference to git username and password
type GitCreds struct {
	Username *corev1.SecretKeySelector `json:"username" protobuf:"bytes,1,opt,name=username"`
	Password *corev1.SecretKeySelector `json:"password" protobuf:"bytes,2,opt,name=password"`
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
