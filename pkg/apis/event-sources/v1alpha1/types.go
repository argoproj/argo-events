package v1alpha1

import (
	"encoding/json"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
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

type EventSourceSpec struct {
	Minio map[string]apicommon.S3Artifact `json:"minio,omitempty" protobuf:"bytes,1,opt,name=minio"`

	Calendar map[string]CalendarEventSource `json:"calendar,omitempty" protobuf:"bytes,2,opt,name=calendar"`

	File map[string]FileEventSource `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`

	Resource map[string]ResourceEventSource `json:"resource,omitempty" protobuf:"bytes,4,opt,name=resource"`
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
	Filter *ResourceFilter `json:"filter,omitempty" protobuf:"bytes,2,opt,name=filter"`
	// Group of the resource
	metav1.GroupVersionResource `json:",inline"`
	// Type is the event type.
	// If not provided, the gateway will watch all events for a resource.
	EventType EventType `json:"eventType,omitempty" protobuf:"bytes,3,opt,name=eventType"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource event objects
type ResourceFilter struct {
	Prefix    string            `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	Labels    map[string]string `json:"labels,omitempty" protobuf:"bytes,2,opt,name=labels"`
	Fields    map[string]string `json:"fields,omitempty" protobuf:"bytes,3,opt,name=fields"`
	CreatedBy metav1.Time       `json:"createdBy,omitempty" protobuf:"bytes,4,opt,name=createdBy"`
}

type EventSourceStatus struct {
	CreatedAt metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,1,opt,name=createdAt"`
}
