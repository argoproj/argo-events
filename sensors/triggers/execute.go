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

package triggers

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// FetchResource fetches the K8s resource
func FetchResource(kubeClient kubernetes.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger) (*unstructured.Unstructured, error) {
	if trigger.Template != nil {
		if err := ApplyTemplateParameters(sensor, trigger); err != nil {
			return nil, err
		}
		creds, err := store.GetCredentials(kubeClient, sensor.Namespace, trigger.Template.Source)
		if err != nil {
			return nil, err
		}
		reader, err := store.GetArtifactReader(trigger.Template.Source, creds, kubeClient)
		if err != nil {
			return nil, err
		}
		uObj, err := store.FetchArtifact(reader, trigger.Template.GroupVersionResource)
		if err != nil {
			return nil, err
		}
		return uObj, nil
	}
	return nil, nil
}

func Execute(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, obj *unstructured.Unstructured, client dynamic.NamespaceableResourceInterface) (*unstructured.Unstructured, error) {
	namespace := obj.GetNamespace()
	// Defaults to sensor's namespace
	if namespace == "" {
		namespace = sensor.Namespace
	}
	obj.SetNamespace(namespace)

	return client.Namespace(namespace).Create(obj, metav1.CreateOptions{})
}
