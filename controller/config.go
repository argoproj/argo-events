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

package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/blackrock/axis/common"
	"github.com/ghodss/yaml"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func (c *SensorController) watchControllerConfigMap(ctx context.Context) (cache.Controller, error) {
	c.log.Info("watching sensor controller config map updates")
	source := c.newControllerConfigMapWatch()
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					c.log.Info("detected ConfigMap update. Updating the controller config.")
					err := c.updateConfig(cm)
					if err != nil {
						c.log.Errorf("update of config failed due to: %v", err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					c.log.Info("detected ConfigMap update. Updating the controller config.")
					err := c.updateConfig(newCm)
					if err != nil {
						c.log.Errorf("update of config failed due to: %v", err)
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (c *SensorController) newControllerConfigMapWatch() *cache.ListWatch {
	x := c.kubeClientset.CoreV1().RESTClient()
	resource := "configmaps"
	name := c.ConfigMap
	namespace := c.ConfigMapNS
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// ResyncConfig reloads the controller config from the configmap
func (c *SensorController) ResyncConfig() error {
	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		namespace = common.DefaultSensorControllerNamespace
	}
	cmClient := c.kubeClientset.CoreV1().ConfigMaps(namespace)
	cm, err := cmClient.Get(c.ConfigMap, metav1.GetOptions{})
	if err != nil {
		return err
	}
	c.ConfigMapNS = cm.Namespace
	return c.updateConfig(cm)
}

func (c *SensorController) updateConfig(cm *apiv1.ConfigMap) error {
	configStr, ok := cm.Data[common.SensorControllerConfigMapKey]
	if !ok {
		return fmt.Errorf("configMap '%s' does not have key '%s'", c.ConfigMap, common.SensorControllerConfigMapKey)
	}
	var config SensorControllerConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return err
	}
	c.Config = config
	return nil
}
