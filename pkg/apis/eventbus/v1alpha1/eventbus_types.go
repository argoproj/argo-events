package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/common"
)

// EventBus is the definition of a eventbus resource
// +genclient
// +kubebuilder:resource:singular=eventbus,shortName=eb
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type EventBus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              EventBusSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// +optional
	Status EventBusStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// EventBusList is the list of eventbus resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EventBusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Items []EventBus `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// EventBusSpec refers to specification of eventbus resource
type EventBusSpec struct {
	// NATS eventbus
	// +optional
	NATS *NATSBus `json:"nats,omitempty" protobuf:"bytes,1,opt,name=nats"`
	// +optional
	JetStream *JetStreamBus `json:"jetstream,omitempty" protobuf:"bytes,2,opt,name=jetstream"`
}

// EventBusStatus holds the status of the eventbus resource
type EventBusStatus struct {
	common.Status `json:",inline" protobuf:"bytes,1,opt,name=status"`
	// Config holds the fininalized configuration of EventBus
	Config BusConfig `json:"config,omitempty" protobuf:"bytes,2,opt,name=config"`
}

// BusConfig has the finalized configuration for EventBus
type BusConfig struct {
	// +optional
	NATS *NATSConfig `json:"nats,omitempty" protobuf:"bytes,1,opt,name=nats"`
	// +optional
	JetStream *JetStreamConfig `json:"jetstream,omitempty" protobuf:"bytes,2,opt,name=jetstream"`
	// +optional (TODO)
	Kafka *interface{} `json:"kafka,omitempty" protobuf:"bytes,2,opt,name=kafka"`
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
	s.InitializeConditions(EventBusConditionDeployed, EventBusConditionConfigured)
}

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
