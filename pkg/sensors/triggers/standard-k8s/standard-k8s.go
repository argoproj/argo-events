/*
Copyright 2020 The Argoproj Authors.

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
	"context"
	"fmt"
	"strconv"
	"time"

	"dario.cat/mergo"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/sensors/policy"
	"github.com/argoproj/argo-events/pkg/sensors/triggers"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

var clusterResources = map[string]bool{
	"namespaces":          true,
	"nodes":               true,
	"persistentvolumes":   true,
	"clusterroles":        true,
	"clusterrolebindings": true,
}

// StandardK8STrigger implements Trigger interface for standard Kubernetes resources
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
	Logger *zap.SugaredLogger

	namespableDynamicClient dynamic.NamespaceableResourceInterface
}

// NewStandardK8sTrigger returns a new StandardK8STrigger
func NewStandardK8sTrigger(k8sClient kubernetes.Interface, dynamicClient dynamic.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) *StandardK8sTrigger {
	return &StandardK8sTrigger{
		K8sClient:     k8sClient,
		DynamicClient: dynamicClient,
		Sensor:        sensor,
		Trigger:       trigger,
		Logger:        logger.With(logging.LabelTriggerType, v1alpha1.TriggerTypeK8s),
	}
}

// GetTriggerType returns the type of the trigger
func (k8sTrigger *StandardK8sTrigger) GetTriggerType() v1alpha1.TriggerType {
	return v1alpha1.TriggerTypeK8s
}

// FetchResource fetches the trigger resource from external source
func (k8sTrigger *StandardK8sTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	trigger := k8sTrigger.Trigger
	var rObj runtime.Object

	uObj, err := triggers.FetchKubernetesResource(trigger.Template.K8s.Source)
	if err != nil {
		return nil, err
	}

	gvr := triggers.GetGroupVersionResource(uObj)
	k8sTrigger.namespableDynamicClient = k8sTrigger.DynamicClient.Resource(gvr)

	if trigger.Template.K8s.LiveObject && trigger.Template.K8s.Operation == v1alpha1.Update {
		objName := uObj.GetName()
		if objName == "" {
			return nil, fmt.Errorf("resource name must be specified for fetching live object")
		}

		objNamespace := uObj.GetNamespace()
		_, isClusterResource := clusterResources[gvr.Resource]
		if objNamespace == "" && !isClusterResource {
			return nil, fmt.Errorf("resource namespace must be specified for fetching live object")
		}
		rObj, err = k8sTrigger.namespableDynamicClient.Namespace(objNamespace).Get(ctx, objName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		rObj = uObj
	}
	return rObj, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (k8sTrigger *StandardK8sTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the trigger resource")
	}
	if err := triggers.ApplyResourceParameters(events, k8sTrigger.Trigger.Template.K8s.Parameters, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// Execute executes the trigger
func (k8sTrigger *StandardK8sTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger := k8sTrigger.Trigger

	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the trigger resource")
	}

	gvr := triggers.GetGroupVersionResource(obj)
	namespace := ""
	if _, isClusterResource := clusterResources[gvr.Resource]; !isClusterResource {
		namespace = obj.GetNamespace()
		// Defaults to sensor's namespace
		if namespace == "" {
			namespace = k8sTrigger.Sensor.Namespace
		}
	}
	obj.SetNamespace(namespace)

	op := v1alpha1.Create
	if trigger.Template.K8s.Operation != "" {
		op = trigger.Template.K8s.Operation
	}

	// We might have a client from FetchResource() already, or we might not have one yet.
	if k8sTrigger.namespableDynamicClient == nil {
		k8sTrigger.namespableDynamicClient = k8sTrigger.DynamicClient.Resource(gvr)
	}

	switch op {
	case v1alpha1.Create:
		k8sTrigger.Logger.Info("creating the object...")
		// Add labels
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["events.argoproj.io/sensor"] = k8sTrigger.Sensor.Name
		labels["events.argoproj.io/trigger"] = trigger.Template.Name
		labels["events.argoproj.io/action-timestamp"] = strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond)))
		obj.SetLabels(labels)
		return k8sTrigger.namespableDynamicClient.Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})

	case v1alpha1.Update:
		k8sTrigger.Logger.Info("updating the object...")

		oldObj, err := k8sTrigger.namespableDynamicClient.Namespace(namespace).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			k8sTrigger.Logger.Info("object not found, creating the object...")
			return k8sTrigger.namespableDynamicClient.Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
		} else if err != nil {
			return nil, fmt.Errorf("failed to retrieve existing object. err: %w", err)
		}

		if err := mergo.Merge(oldObj, obj, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("failed to update the object. err: %w", err)
		}

		return k8sTrigger.namespableDynamicClient.Namespace(namespace).Update(ctx, oldObj, metav1.UpdateOptions{})

	case v1alpha1.Patch:
		k8sTrigger.Logger.Info("patching the object...")

		_, err := k8sTrigger.namespableDynamicClient.Namespace(namespace).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			k8sTrigger.Logger.Info("object not found, creating the object...")
			return k8sTrigger.namespableDynamicClient.Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
		} else if err != nil {
			return nil, fmt.Errorf("failed to retrieve existing object. err: %w", err)
		}

		if k8sTrigger.Trigger.Template.K8s.PatchStrategy == "" {
			k8sTrigger.Trigger.Template.K8s.PatchStrategy = k8stypes.MergePatchType
		}

		body, err := obj.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal object into JSON schema. err: %w", err)
		}

		return k8sTrigger.namespableDynamicClient.Namespace(namespace).Patch(ctx, obj.GetName(), k8sTrigger.Trigger.Template.K8s.PatchStrategy, body, metav1.PatchOptions{})

	case v1alpha1.Delete:
		k8sTrigger.Logger.Info("deleting the object...")
		_, err := k8sTrigger.namespableDynamicClient.Namespace(namespace).Get(ctx, obj.GetName(), metav1.GetOptions{})

		if err != nil && apierrors.IsNotFound(err) {
			k8sTrigger.Logger.Info("object not found, nothing to delete...")
			return nil, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to retrieve existing object. err: %w", err)
		}

		err = k8sTrigger.namespableDynamicClient.Namespace(namespace).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to delete object. err: %w", err)
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown operation type %s", string(op))
	}
}

// ApplyPolicy applies the policy on the trigger
func (k8sTrigger *StandardK8sTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	trigger := k8sTrigger.Trigger

	if trigger.Policy == nil || trigger.Policy.K8s == nil || trigger.Policy.K8s.Labels == nil {
		return nil
	}

	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("failed to interpret the trigger resource")
	}

	p := policy.NewResourceLabels(trigger, k8sTrigger.namespableDynamicClient, obj)
	if p == nil {
		return nil
	}

	err := p.ApplyPolicy(ctx)
	if err != nil {
		if wait.Interrupted(err) {
			if trigger.Policy.K8s.ErrorOnBackoffTimeout {
				return fmt.Errorf("failed to determine status of the triggered resource. setting trigger state as failed")
			}
			return nil
		} else {
			return err
		}
	}
	return nil
}
