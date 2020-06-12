package v1alpha1

import (
	"reflect"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventBus is the definition of a eventbus resource
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type EventBus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              EventBusSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            EventBusStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// EventBusList is the list of eventbus resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EventBusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	// +listType=eventbus
	Items []EventBus `json:"items" protobuf:"bytes,2,opt,name=items"`
}

// EventBusSpec refers to specification of eventbus resource
type EventBusSpec struct {
	// NATS eventbus
	NATS *NATSBus `json:"nats,omitempty" protobuf:"bytes,1,opt,name=nats"`
}

// EventBusStatus holds the status of the eventbus resource
type EventBusStatus struct {
	Conditions []Condition `json:"conditions,omitempty" protobuf:"bytes,1,rep,name=conditions"`
	// Config holds the fininalized configuration of EventBus
	Config BusConfig `json:"config,omitempty" protobuf:"bytes,2,opt,name=config"`
}

// NATSBus holds the NATS eventbus information
type NATSBus struct {
	// Native means to bring up a native NATS service
	Native *NativeStrategy `json:"native,omitempty" protobuf:"bytes,1,opt,name=native"`
	// Exotic holds an exotic NATS config
	Exotic *NATSConfig `json:"exotic,omitempty" protobuf:"bytes,2,opt,name=exotic"`
}

// AuthStrategy is the auth strategy of native nats installaion
type AuthStrategy string

// possible auth strategies
var (
	AuthStrategyNone  AuthStrategy = "none"
	AuthStrategyToken AuthStrategy = "token"
)

// NativeStrategy indicates to install a native NATS service
type NativeStrategy struct {
	// Replicas is the NATS StatefulSet size
	Replicas     int32         `json:"replicas,omitempty" protobuf:"bytes,1,opt,name=replicas"`
	Auth         *AuthStrategy `json:"auth,omitempty" protobuf:"bytes,2,opt,name=auth"`
	AntiAffinity bool          `json:"antiAffinity,omitempty" protobuf:"bytes,3,opt,name=antiAffinity"`
	// +optional
	Persistence *PersistenceStrategy `json:"persistence,omitempty" protobuf:"bytes,4,opt,name=persistence"`
}

// PersistenceStrategy defines the strategy of persistence
type PersistenceStrategy struct {
	// Name of the StorageClass required by the claim.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty" protobuf:"bytes,1,opt,name=storageClassName"`
	// Available access modes such as ReadWriteOnce, ReadWriteMany
	// https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes
	// +optional
	AccessMode *corev1.PersistentVolumeAccessMode `json:"accessMode,omitempty" protobuf:"bytes,2,opt,name=accessMode"`
	// Volume size, e.g. 10Gi
	VolumeSize *apiresource.Quantity `json:"volumeSize,omitempty" protobuf:"bytes,3,opt,name=volumeSize"`
}

// BusConfig has the finalized configuration for EventBus
type BusConfig struct {
	NATS *NATSConfig `json:"nats,omitempty" protobuf:"bytes,1,opt,name=nats"`
}

// NATSConfig holds the config of NATS
type NATSConfig struct {
	// NATS host url
	URL string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	// Cluster ID for nats streaming, if it's missing, treat it as NATS server
	// +optional
	ClusterID *string `json:"clusterID,omitempty" protobuf:"bytes,2,opt,name=clusterID"`
	// Auth strategy, default to AuthStrategyNone
	// +optional
	Auth *AuthStrategy `json:"auth,omitempty" protobuf:"bytes,3,opt,name=auth"`
	// Secret for auth
	// +optional
	AccessSecret *corev1.SecretKeySelector `json:"accessSecret,omitempty" protobuf:"bytes,4,opt,name=accessSecret"`
}

const (
	// EventBusConditionDeployed has the status True when the EventBus
	// has its RestfulSet/Deployment ans service created.
	EventBusConditionDeployed ConditionType = "Deployed"
	// EventBusConditionConfigured has the status True when the EventBus
	// has its configuration ready.
	EventBusConditionConfigured ConditionType = "Configured"
)

// MarkDeployed set the bus has been deployed.
func (s *EventBusStatus) MarkDeployed(reason, message string) {
	s.MarkTrueWithReason(EventBusConditionDeployed, reason, message)
}

// MarkDeploying set the bus is deploying
func (s *EventBusStatus) MarkDeploying(reason, message string) {
	s.MarkUnknown(EventBusConditionDeployed, reason, message)
}

// MarkDeployFailed set the bus deploy failed
func (s *EventBusStatus) MarkDeployFailed(reason, message string) {
	s.MarkFalse(EventBusConditionDeployed, reason, message)
}

// MarkConfigured set the bus configuration has been done.
func (s *EventBusStatus) MarkConfigured() {
	s.MarkTrue(EventBusConditionConfigured)
}

// MarkNotConfigured set the bus status not configured.
func (s *EventBusStatus) MarkNotConfigured(reason, message string) {
	s.MarkFalse(EventBusConditionConfigured, reason, message)
}

// ConditionType is a valid value of Condition.Type
type ConditionType string

// Condition contains details about resource state
type Condition struct {
	// Condition type.
	// +required
	Type ConditionType `json:"type" protobuf:"bytes,1,opt,name=type" protobuf:"bytes,1,opt,name=type"`
	// Condition status, True, False or Unknown.
	// +required
	Status corev1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// Unique, this should be a short, machine understandable string that gives the reason
	// for condition's last transition. For example, "ImageNotFound"
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// IsTrue tells if the condition is True
func (c *Condition) IsTrue() bool {
	if c == nil {
		return false
	}
	return c.Status == corev1.ConditionTrue
}

// IsFalse tells if the condition is False
func (c *Condition) IsFalse() bool {
	if c == nil {
		return false
	}
	return c.Status == corev1.ConditionFalse
}

// IsUnknown tells if the condition is Unknown
func (c *Condition) IsUnknown() bool {
	if c == nil {
		return true
	}
	return c.Status == corev1.ConditionUnknown
}

// GetReason returns as Reason
func (c *Condition) GetReason() string {
	if c == nil {
		return ""
	}
	return c.Reason
}

// GetMessage returns a Message
func (c *Condition) GetMessage() string {
	if c == nil {
		return ""
	}
	return c.Message
}

// InitConditions initializes the contions to Unknown
func (s *EventBusStatus) InitConditions(conditionTypes ...ConditionType) {
	for _, t := range conditionTypes {
		c := Condition{
			Type:   t,
			Status: corev1.ConditionUnknown,
		}
		s.SetCondition(c)
	}
}

// SetCondition sets a condition
func (s *EventBusStatus) SetCondition(condition Condition) {
	var conditions []Condition
	for _, c := range s.Conditions {
		if c.Type != condition.Type {
			conditions = append(conditions, c)
		} else {
			condition.LastTransitionTime = c.LastTransitionTime
			if reflect.DeepEqual(&condition, &c) {
				return
			}
		}
	}
	condition.LastTransitionTime = metav1.NewTime(time.Now())
	conditions = append(conditions, condition)
	// Sort for easy read
	sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
	s.Conditions = conditions
}

func (s *EventBusStatus) markTypeStatus(t ConditionType, status corev1.ConditionStatus, reason, message string) {
	s.SetCondition(Condition{
		Type:    t,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

// MarkTrue sets the status of t to true
func (s *EventBusStatus) MarkTrue(t ConditionType) {
	s.markTypeStatus(t, corev1.ConditionTrue, "", "")
}

// MarkTrueWithReason sets the status of t to true with reason
func (s *EventBusStatus) MarkTrueWithReason(t ConditionType, reason, message string) {
	s.markTypeStatus(t, corev1.ConditionTrue, reason, message)
}

// MarkFalse sets the status of t to fasle
func (s *EventBusStatus) MarkFalse(t ConditionType, reason, message string) {
	s.markTypeStatus(t, corev1.ConditionFalse, reason, message)
}

// MarkUnknown sets the status of t to unknown
func (s *EventBusStatus) MarkUnknown(t ConditionType, reason, message string) {
	s.markTypeStatus(t, corev1.ConditionUnknown, reason, message)
}

// GetCondition returns the condition of a condtion type
func (s *EventBusStatus) GetCondition(t ConditionType) *Condition {
	for _, c := range s.Conditions {
		if c.Type == t {
			return &c
		}
	}
	return nil
}

// IsReady returns true when all the conditions are true
func (s *EventBusStatus) IsReady() bool {
	if len(s.Conditions) == 0 {
		return false
	}
	for _, c := range s.Conditions {
		if !c.IsTrue() {
			return false
		}
	}
	return true
}
