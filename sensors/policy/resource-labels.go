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
	"fmt"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
)

type ResourceLabels struct {
	Trigger           *v1alpha1.Trigger
	resourceInterface dynamic.NamespaceableResourceInterface
	Name              string
	namespace         string
}

func (rl *ResourceLabels) ApplyPolicy() error {
	err := wait.ExponentialBackoff(wait.Backoff{
		Duration: rl.Trigger.Policy.Backoff.Duration,
		Steps:    rl.Trigger.Policy.Backoff.Steps,
		Factor:   rl.Trigger.Policy.Backoff.Factor,
		Jitter:   rl.Trigger.Policy.Backoff.Jitter,
	}, func() (bool, error) {
		obj, err := rl.resourceInterface.Namespace(rl.namespace).Get(rl.Name, metav1.GetOptions{})
		if err != nil {
			sensorCtx.logger.WithError(err).WithField("resource-name", obj.GetName()).Error("failed to get triggered resource")
			return false, nil
		}

		labels := obj.GetLabels()
		if labels == nil {
			sensorCtx.logger.Warn("triggered object does not have labels, won't apply the trigger policy")
			return false, nil
		}
		sensorCtx.logger.WithField("labels", labels).Debug("object labels")

		// check if success labels match with labels on object
		if trigger.Policy.State.Success != nil {
			success := true
			for successKey, successValue := range trigger.Policy.State.Success {
				if value, ok := labels[successKey]; ok {
					if successValue != value {
						success = false
						break
					}
					continue
				}
				success = false
			}
			if success {
				return true, nil
			}
		}

		// check if failure labels match with labels on object
		if trigger.Policy.State.Failure != nil {
			failure := true
			for failureKey, failureValue := range trigger.Policy.State.Failure {
				if value, ok := labels[failureKey]; ok {
					if failureValue != value {
						failure = false
						break
					}
					continue
				}
				failure = false
			}
			if failure {
				return false, fmt.Errorf("trigger is in failed state")
			}
		}

		return false, nil
	})

	return err
}
