/*
Copyright 2018 The Argoproj Authors.

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

package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// SetObjectMeta sets ObjectMeta of child resource
func SetObjectMeta(owner, obj metav1.Object, gvk schema.GroupVersionKind) error {
	references := obj.GetOwnerReferences()
	references = append(references,
		*metav1.NewControllerRef(owner, gvk),
	)
	obj.SetOwnerReferences(references)

	if obj.GetName() == "" && obj.GetGenerateName() == "" {
		obj.SetName(owner.GetName())
	}
	if obj.GetNamespace() == "" {
		obj.SetNamespace(owner.GetNamespace())
	}

	hash, err := sharedutil.GetObjectHash(obj)
	if err != nil {
		return err
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[v1alpha1.AnnotationResourceSpecHash] = hash
	obj.SetAnnotations(annotations)

	return nil
}
