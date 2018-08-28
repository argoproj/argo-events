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
// +genclient:noStatus
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
	// Image is the gateway processor image
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`

	// ImagePullPolicy for pulling the image
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty" protobuf:"bytes,10,opt,name=imagePullPolicy,casttype=k8s.io/api/core/v1.PullPolicy"`

	// Command is command to run gateway processor image
	Command string `json:"command" protobuf:"bytes,2,opt,name=command"`

	// Todo: does this needed to specified separately?
	// ConfigMap is name of the configmap gateway processor can access if required
	ConfigMap string `json:"configMap,omitempty" protobuf:"bytes,3,opt,name=configmap"`

	// Type is type of the gateway
	Type string `json:"type" protobuf:"bytes,5,opt,name=type"`

	// Version is used for marking event version
	Version string `json:"version" protobuf:"bytes,6,opt,name=version"`

	// Service is the name of the service to expose the gateway
	Service Service `json:"service,omitempty" protobuf:"bytes,7,opt,name=service"`

	// Sensors are list of sensors to dispatch events to
	Sensors []string `json:"sensors" protobuf:"bytes,8,opt,name=sensors"`

	// ServiceAccountName is name of service account to run the gateway
	ServiceAccountName string `json:"serviceAccountName" protobuf:"bytes,9,opt,name=serviceAccountName"`
}

// NodePhase is the label for the condition of a node
type NodePhase string

// possible types of node phases
const (
	NodePhaseRunning NodePhase = "Running" // the node is running
	NodePhaseError   NodePhase = "Error"   // the node has encountered an error in processing
	NodePhaseNew     NodePhase = ""        // the node is new
)

// GatewayStatus contains information about the status of a gateway.
type GatewayStatus struct {
	// Phase is the high-level summary of the gateway
	Phase NodePhase `json:"phase" protobuf:"bytes,1,opt,name=phase"`

	// StartedAt is the time at which this gateway was initiated
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`

	// Message is a human readable string indicating details about a gateway in its phase
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
}

// Service exposed gateway to outside cluster or in cluster components depending on it's type.
type Service struct {
	// Type is type of the service. Either ClusterIP, NodePort, LoadBalancer or ExternalName
	// See https://kubernetes.io/docs/concepts/services-networking/service/
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`

	// Port is port exposed to components outside cluster
	Port int32 `json:"port" protobuf:"bytes,2,opt,name=port"`

	// TargetPort is the gateway http server port
	TargetPort int32 `json:"targetPort" protobuf:"bytes,3,opt,name=targetPort"`
}
