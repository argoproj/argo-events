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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Items           []Gateway `json:"items" protobuf:"bytes,2,opt,name=items"`
}

// GatewaySpec represents gateway specifications
type GatewaySpec struct {
	// DeploySpec is description of gateway
	DeploySpec *corev1.PodSpec `json:"deploySpec" protobuf:"bytes,1,opt,name=deploySpec"`

	// ConfigMap is name of the configmap for gateway-processor
	ConfigMap string `json:"configMap,omitempty" protobuf:"bytes,2,opt,name=configmap"`

	// Type is type of the gateway
	Type string `json:"type" protobuf:"bytes,3,opt,name=type"`

	// Version is used for marking event version
	Version string `json:"version" protobuf:"bytes,4,opt,name=version"`

	// ServiceSpec is the specifications of the service to expose the gateway
	ServiceSpec *corev1.ServiceSpec `json:"serviceSpec,omitempty" protobuf:"bytes,5,opt,name=serviceSpec"`

	// Sensors are list of sensors to dispatch events to
	Sensors []string `json:"sensors" protobuf:"bytes,6,opt,name=sensors"`

	// RPCPort if provided is used to communicate between gRPC gateway client and gRPC gateway server
	RPCPort string `json:"rpcPort,omitempty" protobuf:"bytes,7,opt,name=rpcPort"`

	// HTTPServerPort if provided is used to communicate between gateway client and server over http
	HTTPServerPort string `json:"httpServerPort,omitempty" protobuf:"bytes,8,opt,name=httpServerPort"`
}

// NodePhase is the label for the condition of a node
type NodePhase string

// possible types of node phases
const (
	NodePhaseRunning      NodePhase = "Running"      // the node is running
	NodePhaseError        NodePhase = "Error"        // the node has encountered an error in processing
	NodePhaseNew          NodePhase = ""             // the node is new
	NodePhaseServiceError NodePhase = "ServiceError" // failed to expose gateway using a service
	NodePhaseCompleted    NodePhase = "Completed"    // node has completed running
	NodePhaseInitialized  NodePhase = "Initialized"  // node has been initialized
)

// GatewayStatus contains information about the status of a gateway.
type GatewayStatus struct {
	// Phase is the high-level summary of the gateway
	Phase NodePhase `json:"phase" protobuf:"bytes,1,opt,name=phase"`

	// StartedAt is the time at which this gateway was initiated
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`

	// Message is a human readable string indicating details about a gateway in its phase
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// Nodes is a mapping between a node ID and the node's status
	// it records the states for the configurations of gateway.
	Nodes map[string]NodeStatus `json:"nodes,omitempty" protobuf:"bytes,5,rep,name=nodes"`
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
	Phase NodePhase `json:"phase" protobuf:"bytes,4,opt,name=phase"`

	// StartedAt is the time at which this node started
	StartedAt metav1.MicroTime `json:"startedAt,omitempty" protobuf:"bytes,5,opt,name=startedAt"`

	// Message store data or something to save for configuration
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`

	UpdateTime metav1.MicroTime `json:"updateTime,omitempty" protobuf:"bytes,7,opt,name=updateTime"`
}
