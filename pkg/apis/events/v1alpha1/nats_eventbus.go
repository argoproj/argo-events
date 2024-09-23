package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

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
	AuthStrategyBasic AuthStrategy = "basic"
)

// NativeStrategy indicates to install a native NATS service
type NativeStrategy struct {
	// Size is the NATS StatefulSet size
	Replicas int32         `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	Auth     *AuthStrategy `json:"auth,omitempty" protobuf:"bytes,2,opt,name=auth,casttype=AuthStrategy"`
	// +optional
	Persistence *PersistenceStrategy `json:"persistence,omitempty" protobuf:"bytes,3,opt,name=persistence"`
	// ContainerTemplate contains customized spec for NATS container
	// +optional
	ContainerTemplate *ContainerTemplate `json:"containerTemplate,omitempty" protobuf:"bytes,4,opt,name=containerTemplate"`
	// MetricsContainerTemplate contains customized spec for metrics container
	// +optional
	MetricsContainerTemplate *ContainerTemplate `json:"metricsContainerTemplate,omitempty" protobuf:"bytes,5,opt,name=metricsContainerTemplate"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,6,rep,name=nodeSelector"`
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty" protobuf:"bytes,7,rep,name=tolerations"`
	// Metadata sets the pods's metadata, i.e. annotations and labels
	Metadata *Metadata `json:"metadata,omitempty" protobuf:"bytes,8,opt,name=metadata"`
	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,9,opt,name=securityContext"`
	// Max Age of existing messages, i.e. "72h", “4h35m”
	// +optional
	MaxAge *string `json:"maxAge,omitempty" protobuf:"bytes,10,opt,name=maxAge"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,11,rep,name=imagePullSecrets"`
	// ServiceAccountName to apply to NATS StatefulSet
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,12,opt,name=serviceAccountName"`
	// If specified, indicates the EventSource pod's priority. "system-node-critical"
	// and "system-cluster-critical" are two special keywords which indicate the
	// highest priorities with the former being the highest priority. Any other
	// name must be defined by creating a PriorityClass object with that name.
	// If not specified, the pod priority will be default or zero if there is no
	// default.
	// More info: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty" protobuf:"bytes,13,opt,name=priorityClassName"`
	// The priority value. Various system components use this field to find the
	// priority of the EventSource pod. When Priority Admission Controller is enabled,
	// it prevents users from setting this field. The admission controller populates
	// this field from PriorityClassName.
	// The higher the value, the higher the priority.
	// More info: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
	// +optional
	Priority *int32 `json:"priority,omitempty" protobuf:"bytes,14,opt,name=priority"`
	// The pod's scheduling constraints
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty" protobuf:"bytes,15,opt,name=affinity"`
	// Maximum number of messages per channel, 0 means unlimited. Defaults to 1000000
	MaxMsgs *uint64 `json:"maxMsgs,omitempty" protobuf:"bytes,16,opt,name=maxMsgs"`
	// Total size of messages per channel, 0 means unlimited. Defaults to 1GB
	MaxBytes *string `json:"maxBytes,omitempty" protobuf:"bytes,17,opt,name=maxBytes"`
	// Maximum number of subscriptions per channel, 0 means unlimited. Defaults to 1000
	MaxSubs *uint64 `json:"maxSubs,omitempty" protobuf:"bytes,18,opt,name=maxSubs"`
	// Maximum number of bytes in a message payload, 0 means unlimited. Defaults to 1MB
	MaxPayload *string `json:"maxPayload,omitempty" protobuf:"bytes,19,opt,name=maxPayload"`
	// Specifies the time in follower state without a leader before attempting an election, i.e. "72h", “4h35m”. Defaults to 2s
	RaftHeartbeatTimeout *string `json:"raftHeartbeatTimeout,omitempty" protobuf:"bytes,20,opt,name=raftHeartbeatTimeout"`
	// Specifies the time in candidate state without a leader before attempting an election, i.e. "72h", “4h35m”. Defaults to 2s
	RaftElectionTimeout *string `json:"raftElectionTimeout,omitempty" protobuf:"bytes,21,opt,name=raftElectionTimeout"`
	// Specifies how long a leader waits without being able to contact a quorum of nodes before stepping down as leader, i.e. "72h", “4h35m”. Defaults to 1s
	RaftLeaseTimeout *string `json:"raftLeaseTimeout,omitempty" protobuf:"bytes,22,opt,name=raftLeaseTimeout"`
	// Specifies the time without an Apply() operation before sending an heartbeat to ensure timely commit, i.e. "72h", “4h35m”. Defaults to 100ms
	RaftCommitTimeout *string `json:"raftCommitTimeout,omitempty" protobuf:"bytes,23,opt,name=raftCommitTimeout"`
}

// GetReplicas return the replicas of statefulset
func (in *NativeStrategy) GetReplicas() int {
	return int(in.Replicas)
}

// NATSConfig holds the config of NATS
type NATSConfig struct {
	// NATS streaming url
	URL string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	// Cluster ID for nats streaming
	ClusterID *string `json:"clusterID,omitempty" protobuf:"bytes,2,opt,name=clusterID"`
	// Auth strategy, default to AuthStrategyNone
	// +optional
	Auth *AuthStrategy `json:"auth,omitempty" protobuf:"bytes,3,opt,name=auth,casttype=AuthStrategy"`
	// Secret for auth
	// +optional
	AccessSecret *corev1.SecretKeySelector `json:"accessSecret,omitempty" protobuf:"bytes,4,opt,name=accessSecret"`
}
