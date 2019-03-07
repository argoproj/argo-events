package common

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChildContext holds necessary information for child resource setup
type ChildResourceContext struct {
	LabelOwnerName                    string
	LabelKeyOwnerControllerInstanceID string
	AnnotationOwnerResourceHashName   string
	InstanceID                        string
}

// SetObjectMeta sets ObjectMeta of child resource
func (ctx *ChildResourceContext) SetObjectMeta(owner, obj metav1.Object) error {
	references := obj.GetOwnerReferences()
	references = append(references,
		*metav1.NewControllerRef(owner, v1alpha1.SchemaGroupVersionKind),
	)
	obj.SetOwnerReferences(references)

	if obj.GetName() == "" && obj.GetGenerateName() == "" {
		obj.SetGenerateName(owner.GetName())
	}

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[ctx.LabelOwnerName] = owner.GetName()
	labels[ctx.LabelKeyOwnerControllerInstanceID] = ctx.InstanceID
	obj.SetLabels(labels)

	hash, err := common.GetObjectHash(obj)
	if err != nil {
		return err
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[ctx.AnnotationOwnerResourceHashName] = hash
	obj.SetAnnotations(annotations)

	return nil
}
