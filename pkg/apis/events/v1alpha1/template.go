package v1alpha1

import (
	"dario.cat/mergo"
	corev1 "k8s.io/api/core/v1"
)

// Template holds the information of a deployment template
type Template struct {
	// Metadata sets the pods's metadata, i.e. annotations and labels
	Metadata *Metadata `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// ServiceAccountName is the name of the ServiceAccount to use to run sensor pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,2,opt,name=serviceAccountName"`
	// Container is the main container image to run in the sensor pod
	// +optional
	Container *Container `json:"container,omitempty" protobuf:"bytes,3,opt,name=container"`
	// Volumes is a list of volumes that can be mounted by containers in a workflow.
	// +patchStrategy=merge
	// +patchMergeKey=name
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,4,rep,name=volumes"`
	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,5,opt,name=securityContext"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,6,rep,name=nodeSelector"`
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty" protobuf:"bytes,7,rep,name=tolerations"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,8,rep,name=imagePullSecrets"`
	// If specified, indicates the EventSource pod's priority. "system-node-critical"
	// and "system-cluster-critical" are two special keywords which indicate the
	// highest priorities with the former being the highest priority. Any other
	// name must be defined by creating a PriorityClass object with that name.
	// If not specified, the pod priority will be default or zero if there is no
	// default.
	// More info: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty" protobuf:"bytes,9,opt,name=priorityClassName"`
	// The priority value. Various system components use this field to find the
	// priority of the EventSource pod. When Priority Admission Controller is enabled,
	// it prevents users from setting this field. The admission controller populates
	// this field from PriorityClassName.
	// The higher the value, the higher the priority.
	// More info: https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
	// +optional
	Priority *int32 `json:"priority,omitempty" protobuf:"bytes,10,opt,name=priority"`
	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty" protobuf:"bytes,11,opt,name=affinity"`
}

// Container defines customized spec for a container
type Container struct {
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,1,opt,name=resources"`
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty" protobuf:"bytes,2,opt,name=imagePullPolicy,casttype=PullPolicy"`
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty" protobuf:"bytes,3,opt,name=securityContext"`
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty" protobuf:"bytes,4,rep,name=volumeMounts"`
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty" protobuf:"bytes,5,rep,name=env"`
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty" protobuf:"bytes,6,rep,name=envFrom"`
}

// ApplyToContainer updates the Container with the values from the customized container
func (c *Container) ApplyToContainer(cc *corev1.Container) {
	_ = mergo.Merge(&cc.Resources, c.Resources, mergo.WithOverride)
	if cc.SecurityContext == nil {
		cc.SecurityContext = c.SecurityContext
	}
	if cc.ImagePullPolicy == "" {
		cc.ImagePullPolicy = c.ImagePullPolicy
	}
	if len(c.Env) > 0 {
		cc.Env = append(cc.Env, c.Env...)
	}
	if len(c.EnvFrom) > 0 {
		cc.EnvFrom = append(cc.EnvFrom, c.EnvFrom...)
	}
	if len(c.VolumeMounts) > 0 {
		cc.VolumeMounts = append(cc.VolumeMounts, c.VolumeMounts...)
	}
}
