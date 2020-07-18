package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/common"
)

// EventBus is the definition of a eventbus resource
// +genclient
// +kubebuilder:resource:singular=eventbus,shortName=eb
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

	Items []EventBus `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// EventBusSpec refers to specification of eventbus resource
type EventBusSpec struct {
	// NATS eventbus
	NATS *NATSBus `json:"nats,omitempty" protobuf:"bytes,1,opt,name=nats"`
}

// EventBusStatus holds the status of the eventbus resource
type EventBusStatus struct {
	common.Status `json:",inline" protobuf:"bytes,1,opt,name=status"`
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
	Replicas     int32         `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	Auth         *AuthStrategy `json:"auth,omitempty" protobuf:"bytes,2,opt,name=auth,casttype=AuthStrategy"`
	AntiAffinity bool          `json:"antiAffinity,omitempty" protobuf:"varint,3,opt,name=antiAffinity"`
	// +optional
	Persistence *PersistenceStrategy `json:"persistence,omitempty" protobuf:"bytes,4,opt,name=persistence"`
	// ContainerTemplate contains customized spec for NATS container
	// +optional
	ContainerTemplate *ContainerTemplate `json:"containerTemplate,omitempty" protobuf:"bytes,5,opt,name=containerTemplate"`
	// MetricsContainerTemplate contains customized spec for metrics container
	// +optional
	MetricsContainerTemplate *ContainerTemplate `json:"metricsContainerTemplate,omitempty" protobuf:"bytes,6,opt,name=metricsContainerTemplate"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,7,rep,name=nodeSelector"`
}

// ContainerTemplate defines customized spec for a container
type ContainerTemplate struct {
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,1,opt,name=resources"`
}

// GetReplicas return the replicas of statefulset
func (in *NativeStrategy) GetReplicas() int {
	return int(in.Replicas)
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
	AccessMode *corev1.PersistentVolumeAccessMode `json:"accessMode,omitempty" protobuf:"bytes,2,opt,name=accessMode,casttype=k8s.io/api/core/v1.PersistentVolumeAccessMode"`
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
	Auth *AuthStrategy `json:"auth,omitempty" protobuf:"bytes,3,opt,name=auth,casttype=AuthStrategy"`
	// Secret for auth
	// +optional
	AccessSecret *corev1.SecretKeySelector `json:"accessSecret,omitempty" protobuf:"bytes,4,opt,name=accessSecret"`
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
