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
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodePhase is the label for the condition of a node.
type NodePhase string

// possible types of node phases
const (
	NodePhaseRunning   NodePhase = "Running"   // the node is running
	NodePhaseError     NodePhase = "Error"     // the node has encountered an error in processing
	NodePhaseNew       NodePhase = ""          // the node is new
	NodePhaseCompleted NodePhase = "Completed" // node has completed running
	NodePhaseRemove    NodePhase = "Remove"    // stale node
)

// Gateway is the definition of a gateway resource
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Status            GatewayStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	Spec              GatewaySpec   `json:"spec" protobuf:"bytes,3,opt,name=spec"`
}

// GatewayList is the list of Gateway resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	// +listType=items
	Items []Gateway `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// GatewaySpec represents gateway specifications
type GatewaySpec struct {
	// Template is the pod specification for the gateway
	// +optional
	Template Template `json:"template,omitempty" protobuf:"bytes,1,opt,name=template"`
	// EventSourceRef refers to event-source that stores event source configurations for the gateway
	EventSourceRef *EventSourceRef `json:"eventSourceRef,omitempty" protobuf:"bytes,2,opt,name=eventSourceRef"`
	// Type is the type of gateway. Used as metadata.
	Type apicommon.EventSourceType `json:"type" protobuf:"bytes,3,opt,name=type,casttype=github.com/argoproj/argo-events/pkg/apis/common.EventSourceType"`
	// Service is the specifications of the service to expose the gateway
	// +optional
	Service *Service `json:"service,omitempty" protobuf:"bytes,4,opt,name=service"`
	// Subscribers holds the contexts of the subscribers/sinks to send events to.
	// +listType=subscribers
	// +optional
	Subscribers *Subscribers `json:"subscribers,omitempty" protobuf:"bytes,5,opt,name=subscribers"`
	// Port on which the gateway event source processor is running on.
	ProcessorPort string `json:"processorPort" protobuf:"bytes,6,opt,name=processorPort"`
	// Replica is the gateway deployment replicas
	Replica int32 `json:"replica,omitempty" protobuf:"varint,7,opt,name=replica"`
}

// Template holds the information of a Gateway deployment template
type Template struct {
	// Metdata sets the pods's metadata, i.e. annotations and labels
	Metadata Metadata `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// ServiceAccountName is the name of the ServiceAccount to use to run gateway pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,2,opt,name=serviceAccountName"`
	// Container is the main container image to run in the gateway pod
	// +optional
	Container *corev1.Container `json:"container,omitempty" protobuf:"bytes,3,opt,name=container"`
	// Volumes is a list of volumes that can be mounted by containers in a workflow.
	// +patchStrategy=merge
	// +patchMergeKey=name
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,4,rep,name=volumes"`
	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,5,opt,name=securityContext"`
	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty" protobuf:"bytes,6,opt,name=affinity"`
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty" protobuf:"bytes,7,rep,name=tolerations"`
	// Spec holds the gateway deployment spec.
	// DEPRECATED: Use Container instead.
	Spec *corev1.PodSpec `json:"spec,omitempty" protobuf:"bytes,8,opt,name=spec"`
}

// Metadata holds the annotations and labels of a gateway pod
type Metadata struct {
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,1,rep,name=annotations"`
	Labels      map[string]string `json:"labels,omitempty" protobuf:"bytes,2,rep,name=labels"`
}

// Service holds the service information gateway exposes
type Service struct {
	// The list of ports that are exposed by this ClusterIP service.
	// +patchMergeKey=port
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=port
	// +listMapKey=protocol
	Ports []corev1.ServicePort `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"port" protobuf:"bytes,1,rep,name=ports"`
	// clusterIP is the IP address of the service and is usually assigned
	// randomly by the master. If an address is specified manually and is not in
	// use by others, it will be allocated to the service; otherwise, creation
	// of the service will fail. This field can not be changed through updates.
	// Valid values are "None", empty string (""), or a valid IP address. "None"
	// can be specified for headless services when proxying is not required.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies
	// +optional
	ClusterIP string `json:"clusterIP,omitempty" protobuf:"bytes,2,opt,name=clusterIP"`
	// Spec holds the gateway service spec.
	// DEPRECATED: Use Ports to declare the ports to be exposed.
	Spec *corev1.ServiceSpec `json:"spec,omitempty" protobuf:"bytes,3,opt,name=spec"`
}

type Subscribers struct {
	// HTTP subscribers are HTTP endpoints to send events to.
	// +listType=string
	// +optional
	HTTP []string `json:"http,omitempty" protobuf:"bytes,1,rep,name=http"`
	// NATS refers to the subscribers over NATS protocol.
	// +listType=NATSSubscriber
	// +optional
	NATS []NATSSubscriber `json:"nats,omitempty" protobuf:"bytes,2,rep,name=nats"`
}

// NATSSubscriber holds the context of subscriber over NATS.
type NATSSubscriber struct {
	// ServerURL refers to the NATS server URL.
	ServerURL string `json:"serverURL" protobuf:"bytes,1,opt,name=serverURL"`
	// Subject refers to the NATS subject name.
	Subject string `json:"subject" protobuf:"bytes,2,opt,name=subject"`
	// Name of the subscription. Must be unique.
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`
}

// EventSourceRef holds information about the EventSourceRef custom resource
type EventSourceRef struct {
	// Name of the event source
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Namespace of the event source
	// Default value is the namespace where referencing gateway is deployed
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
}

// GatewayResource holds the metadata about the gateway resources
type GatewayResource struct {
	// Metadata of the deployment for the gateway
	Deployment *metav1.ObjectMeta `json:"deployment" protobuf:"bytes,1,opt,name=deployment"`
	// Metadata of the service for the gateway
	// +optional
	Service *metav1.ObjectMeta `json:"service,omitempty" protobuf:"bytes,2,opt,name=service"`
}

// GatewayStatus contains information about the status of a gateway.
type GatewayStatus struct {
	// Phase is the high-level summary of the gateway
	Phase NodePhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=NodePhase"`
	// StartedAt is the time at which this gateway was initiated
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`
	// Message is a human readable string indicating details about a gateway in its phase
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
	// Nodes is a mapping between a node ID and the node's status
	// it records the states for the configurations of gateway.
	Nodes map[string]NodeStatus `json:"nodes,omitempty" protobuf:"bytes,4,rep,name=nodes"`
	// Resources refers to the metadata about the gateway resources
	Resources *GatewayResource `json:"resources" protobuf:"bytes,5,opt,name=resources"`
}

// NodeStatus describes the status for an individual node in the gateway configurations.
// A single node can represent one configuration.
type NodeStatus struct {
	// ID is a unique identifier of a node within a sensor
	// It is a hash of the node name
	ID string `json:"id" protobuf:"bytes,1,opt,name=id"`
	// Name is a unique name in the node tree used to generate the node ID
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// DisplayName is the human readable representation of the node
	DisplayName string `json:"displayName" protobuf:"bytes,3,opt,name=displayName"`
	// Phase of the node
	Phase NodePhase `json:"phase" protobuf:"bytes,4,opt,name=phase,casttype=NodePhase"`
	// StartedAt is the time at which this node started
	// +k8s:openapi-gen=false
	StartedAt metav1.MicroTime `json:"startedAt,omitempty" protobuf:"bytes,5,opt,name=startedAt"`
	// Message store data or something to save for configuration
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
	// UpdateTime is the time when node(gateway configuration) was updated
	UpdateTime metav1.MicroTime `json:"updateTime,omitempty" protobuf:"bytes,7,opt,name=updateTime"`
}
