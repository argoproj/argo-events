/*
Copyright 2020 BlackRock, Inc.

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

package standard_k8s

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/argoproj/argo-events/store"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// StandardK8sTrigger implements Trigger interface for standard Kubernetes resources
type StandardK8sTrigger struct {
	// K8sClient is kubernetes client
	K8sClient kubernetes.Interface
	// Dynamic client is Kubernetes dymalic client
	DynamicClient dynamic.Interface
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger definition
	Trigger *v1alpha1.Trigger
	// logger to log stuff
	Logger *logrus.Logger

	namespableDynamicClient dynamic.NamespaceableResourceInterface
}

// NewStandardK8sTrigger returns a new StandardK8sTrigger
func NewStandardK8sTrigger(k8sClient kubernetes.Interface, dynamicClient dynamic.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger) *StandardK8sTrigger {
	return &StandardK8sTrigger{
		K8sClient:     k8sClient,
		DynamicClient: dynamicClient,
		Sensor:        sensor,
		Trigger:       trigger,
		Logger:        logger,
	}
}

// FetchResource fetches the trigger resource from external source
func (k8sTrigger *StandardK8sTrigger) FetchResource() (interface{}, error) {
	trigger := k8sTrigger.Trigger
	if trigger.Template.K8s.Source == nil {
		return nil, errors.Errorf("trigger source for k8s is empty")
	}
	creds, err := store.GetCredentials(k8sTrigger.K8sClient, k8sTrigger.Sensor.Namespace, trigger.Template.K8s.Source)
	if err != nil {
		return nil, err
	}
	reader, err := store.GetArtifactReader(trigger.Template.K8s.Source, creds, k8sTrigger.K8sClient)
	if err != nil {
		return nil, err
	}
	uObj, err := store.FetchArtifact(reader, trigger.Template.K8s.GroupVersionResource)
	if err != nil {
		return nil, err
	}
	return uObj, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (k8sTrigger *StandardK8sTrigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}
	if err := triggers.ApplyResourceParameters(sensor, k8sTrigger.Trigger.Template.K8s.Parameters, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// Execute executes the trigger
func (k8sTrigger *StandardK8sTrigger) Execute(resource interface{}) (interface{}, error) {
	trigger := k8sTrigger.Trigger

	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}

	k8sTrigger.namespableDynamicClient = k8sTrigger.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    trigger.Template.K8s.GroupVersionResource.Group,
		Version:  trigger.Template.K8s.GroupVersionResource.Version,
		Resource: trigger.Template.K8s.GroupVersionResource.Resource,
	})

	namespace := obj.GetNamespace()
	// Defaults to sensor's namespace
	if namespace == "" {
		namespace = k8sTrigger.Sensor.Namespace
	}
	obj.SetNamespace(namespace)

	op := v1alpha1.Create
	if trigger.Template.K8s.Operation != "" {
		op = trigger.Template.K8s.Operation
	}

	switch op {
	case v1alpha1.Create:
		return k8sTrigger.namespableDynamicClient.Namespace(namespace).Create(obj, metav1.CreateOptions{})
	case v1alpha1.Update:
		return k8sTrigger.namespableDynamicClient.Namespace(namespace).Update(obj, metav1.UpdateOptions{})
	default:
		return nil, errors.Errorf("unknown operation type %s", string(op))
	}
}

// ApplyPolicy applies the policy on the trigger
func (k8sTrigger *StandardK8sTrigger) ApplyPolicy(resource interface{}) error {
	trigger := k8sTrigger.Trigger

	if trigger.Policy == nil || trigger.Policy.K8s == nil || trigger.Policy.K8s.Labels == nil {
		return nil
	}

	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return errors.New("failed to interpret the trigger resource")
	}

	p := policy.NewResourceLabels(trigger, k8sTrigger.namespableDynamicClient, obj)
	if p == nil {
		return nil
	}

	err := p.ApplyPolicy()
	if err != nil {
		switch err {
		case wait.ErrWaitTimeout:
			if trigger.Policy.K8s.ErrorOnBackoffTimeout {
				return errors.Errorf("failed to determine status of the triggered resource. setting trigger state as failed")
			}
			return nil
		default:
			return err
		}
	}

	return nil
}
