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

package policy

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// ResourceLabels implements trigger policy based on the resource labels
type ResourceLabels struct {
	Trigger *v1alpha1.Trigger
	Client  dynamic.NamespaceableResourceInterface
	Obj     *unstructured.Unstructured
}

func (rl *ResourceLabels) ApplyPolicy(ctx context.Context) error {
	from := rl.Trigger.Policy.K8s.Backoff
	if rl.Trigger.Policy.K8s == nil || rl.Trigger.Policy.K8s.Labels == nil {
		return nil
	}

	// check if success labels match with labels on object
	completed := false

	backoff := wait.Backoff{
		Duration: *from.Duration,
		Steps:    int(from.Steps),
	}
	if from.Factor != nil {
		backoff.Factor, _ = from.Factor.Float64()
	}
	if from.Jitter != nil {
		jitter, _ := from.Jitter.Float64()
		backoff.Jitter = jitter
	}
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		obj, err := rl.Client.Namespace(rl.Obj.GetNamespace()).Get(ctx, rl.Obj.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		labels := obj.GetLabels()
		if labels == nil {
			return false, nil
		}

		completed = true

		for key, value := range rl.Trigger.Policy.K8s.Labels {
			if v, ok := labels[key]; ok {
				if value != v {
					completed = false
					break
				}
				continue
			}
			completed = false
		}

		if completed {
			return true, nil
		}
		return false, nil
	})

	return err
}

func NewResourceLabels(trigger *v1alpha1.Trigger, client dynamic.NamespaceableResourceInterface, obj *unstructured.Unstructured) *ResourceLabels {
	return &ResourceLabels{
		Trigger: trigger,
		Client:  client,
		Obj:     obj,
	}
}
