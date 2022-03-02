package v1alpha1

import (
	"github.com/argoproj/argo-events/pkg/apis/common"
	corev1 "k8s.io/api/core/v1"
)

// JetStreamBus holds the JetStream EventBus information
type JetStreamBus struct {
	// JetStream version, such as "2.7.3"
	Version string `json:"version,omitempty" protobuf:"bytes,1,opt,name=version"`
	// Redis StatefulSet size
	// +kubebuilder:default=3
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`
	// ContainerTemplate contains customized spec for Nats JetStream container
	// +optional
	ContainerTemplate *ContainerTemplate `json:"containerTemplate,omitempty" protobuf:"bytes,3,opt,name=containerTemplate"`
	// MetricsContainerTemplate contains customized spec for metrics container
	// +optional
	MetricsContainerTemplate *ContainerTemplate `json:"metricsContainerTemplate,omitempty" protobuf:"bytes,4,opt,name=metricsContainerTemplate"`
	// +optional
	Persistence *PersistenceStrategy `json:"persistence,omitempty" protobuf:"bytes,5,opt,name=persistence"`
	// Metadata sets the pods's metadata, i.e. annotations and labels
	Metadata *common.Metadata `json:"metadata,omitempty" protobuf:"bytes,6,opt,name=metadata"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,7,rep,name=nodeSelector"`
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty" protobuf:"bytes,8,rep,name=tolerations"`
	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,9,opt,name=securityContext"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,10,rep,name=imagePullSecrets"`
	// If specified, indicates the Redis pod's priority. "system-node-critical"
	// and "system-cluster-critical" are two special keywords which indicate the
	// highest priorities with the former being the highest priority. Any other
	// name must be defined by creating a PriorityClass object with that name.
	// If not specified, the pod priority will be default or zero if there is no
	// default.
	// More info: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty" protobuf:"bytes,11,opt,name=priorityClassName"`
	// The priority value. Various system components use this field to find the
	// priority of the Redis pod. When Priority Admission Controller is enabled,
	// it prevents users from setting this field. The admission controller populates
	// this field from PriorityClassName.
	// The higher the value, the higher the priority.
	// More info: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
	// +optional
	Priority *int32 `json:"priority,omitempty" protobuf:"bytes,12,opt,name=priority"`
	// The pod's scheduling constraints
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty" protobuf:"bytes,13,opt,name=affinity"`
	// ServiceAccountName to apply to the StatefulSet
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,14,opt,name=serviceAccountName"`
	// JetStream configuration, if not specified, global settings in controller-config will be used.
	// See https://docs.nats.io/running-a-nats-service/configuration#jetstream.
	// Only configure "max_memory_store" or "max_file_store", do not set "store_dir" as it has been hardcoded.
	// +optional
	Settings *string `json:"settings,omitempty" protobuf:"bytes,15,opt,name=settings"`
}

func (j JetStreamBus) GetReplicas() int {
	if j.Replicas == nil {
		return 3
	}
	if *j.Replicas < 3 {
		return 3
	}
	return int(*j.Replicas)
}

type JetStreamConfig struct {
	// JetStream (Nats) URL
	URL  string         `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	Auth *JetStreamAuth `json:"auth,omitempty" protobuf:"bytes,2,opt,name=auth"`
}

type JetStreamAuth struct {
	// Secret for auth token
	// +optional
	Token *corev1.SecretKeySelector `json:"token,omitempty" protobuf:"bytes,1,opt,name=token"`
}
