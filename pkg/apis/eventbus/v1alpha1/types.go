package v1alpha1

import (
	"github.com/argoproj/argo-events/pkg/apis/common"
	corev1 "k8s.io/api/core/v1"
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
	Status common.Status `json:"status,omitempty" protobuf:"bytes,1,opt,name=status"`
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
	// Size is the NATS StatefulSet size
	Size        int                  `json:"size,omitempty" protobuf:"bytes,1,opt,name=size"`
	Auth        *AuthStrategy        `json:"auth,omitempty" protobuf:"bytes,2,opt,name=auth"`
	Persistence *PersistenceStrategy `json:"persistence,omitempty" protobuf:"bytes,3,opt,name=persistence"`
}

// PersistenceStrategy defines the strategy of persistence
type PersistenceStrategy struct {
}

// BusConfig has the finalized configuration for EventBus
type BusConfig struct {
	NATS *NATSConfig `json:"nats,omitempty" protobuf:"bytes,1,opt,name=nats"`
}

// NATSConfig holds the config of NATS
type NATSConfig struct {
	URL          string                   `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	ClusterID    string                   `json:"clusterID,omitempty" protobuf:"bytes,1,opt,name=clusterID"`
	Auth         AuthStrategy             `json:"auth,omitempty" protobuf:"bytes,2,opt,name=auth"`
	AccessSecret corev1.SecretKeySelector `json:"accessSecret,omitempty" protobuf:"bytes,3,opt,name=accessSecret"`
}

const (
	// EventBusConditionDeployed has the status True when the EventBus
	// has its RestfulSet/Deployment ans service created.
	EventBusConditionDeployed common.ConditionType = "Deployed"
	// EventBusConditionConfigured has the status True when the EventBus
	// has its configuration ready.
	EventBusConditionConfigured common.ConditionType = "Configured"
)

// InitConditions sets conditions to Unknown state.
func (s *EventBusStatus) InitConditions() {
	s.Status.InitConditions(EventBusConditionDeployed, EventBusConditionConfigured)
}

// MarkDeployed set the bus has been deployed.
func (s *EventBusStatus) MarkDeployed(reason, message string) {
	s.Status.MarkTrueWithReason(EventBusConditionDeployed, reason, message)
}

// MarkDeploying set the bus is deploying
func (s *EventBusStatus) MarkDeploying(reason, message string) {
	s.Status.MarkUnknown(EventBusConditionDeployed, reason, message)
}

// MarkDeployFailed set the bus deploy failed
func (s *EventBusStatus) MarkDeployFailed(reason, message string) {
	s.Status.MarkFalse(EventBusConditionDeployed, reason, message)
}

// MarkConfigured set the bus configuration has been done.
func (s *EventBusStatus) MarkConfigured() {
	s.Status.MarkTrue(EventBusConditionConfigured)
}

// MarkNotConfigured set the bus status not configured.
func (s *EventBusStatus) MarkNotConfigured(reason, message string) {
	s.Status.MarkFalse(EventBusConditionConfigured, reason, message)
}

func init() {
	SchemeBuilder.Register(&EventBus{}, &EventBusList{})
}
