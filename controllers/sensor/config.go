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

package sensor

import (
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// watchControllerConfigMap watches updates to sensor controller configmap
func (controller *Controller) watchControllerConfigMap() cache.Controller {
	log.Info("watching controller config map updates")
	source := controller.newControllerConfigMapWatch()
	_, ctrl := cache.NewInformer(
		source,
		&corev1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*corev1.ConfigMap); ok {
					log.Info("detected configuration update. updating the controller configuration")
					err := controller.updateConfig(cm)
					if err != nil {
						log.Errorf("update of controller configuration failed due to: %v", err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*corev1.ConfigMap); ok {
					log.Info("detected configuration update. updating the controller configuration")
					err := controller.updateConfig(newCm)
					if err != nil {
						log.Errorf("update of controller configuration failed due to: %v", err)
					}
				}
			},
		})
	return ctrl
}

// newControllerConfigMapWatch returns a configmap watcher
func (controller *Controller) newControllerConfigMapWatch() *cache.ListWatch {
	x := controller.k8sClient.CoreV1().RESTClient()
	resource := "configmaps"
	name := controller.ConfigMap
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(controller.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(controller.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// ResyncConfig reloads the controller config from the configmap
func (controller *Controller) ResyncConfig(namespace string) error {
	cmClient := controller.k8sClient.CoreV1().ConfigMaps(namespace)
	cm, err := cmClient.Get(controller.ConfigMap, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return controller.updateConfig(cm)
}

func (controller *Controller) updateConfig(cm *corev1.ConfigMap) error {
	configStr, ok := cm.Data[common.ControllerConfigMapKey]
	if !ok {
		return errors.Errorf("configMap '%s' does not have key '%s'", controller.ConfigMap, common.ControllerConfigMapKey)
	}
	var config ControllerConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return err
	}
	controller.Config = config
	return nil
}
