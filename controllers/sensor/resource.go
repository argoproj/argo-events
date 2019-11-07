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
	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// sensorResourceLabelSelector returns label selector of the sensor of the context
func (opctx *operationContext) sensorResourceLabelSelector() (labels.Selector, error) {
	req, err := labels.NewRequirement(common.LabelSensorName, selection.Equals, []string{opctx.sensorObj.Name})
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*req), nil
}

// getServiceTemplateSpec returns a K8s service spec for the sensor
func (opctx *operationContext) getServiceTemplateSpec() *pc.ServiceTemplateSpec {
	var serviceSpec *pc.ServiceTemplateSpec
	// Create a ClusterIP service to expose sensor in cluster if the event protocol type is HTTP
	if opctx.sensorObj.Spec.EventProtocol.Type == pc.HTTP {
		serviceSpec = &pc.ServiceTemplateSpec{
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Port:       intstr.Parse(opctx.sensorObj.Spec.EventProtocol.Http.Port).IntVal,
						TargetPort: intstr.FromInt(int(intstr.Parse(opctx.sensorObj.Spec.EventProtocol.Http.Port).IntVal)),
					},
				},
				Type: corev1.ServiceTypeClusterIP,
				Selector: map[string]string{
					common.LabelSensorName:                    opctx.sensorObj.Name,
					common.LabelKeySensorControllerInstanceID: opctx.controller.Config.InstanceID,
				},
			},
		}
	}
	return serviceSpec
}

// buildSensorService returns a new service that exposes sensor.
func (opctx *operationContext) buildSensorService() (*corev1.Service, error) {
	serviceTemplateSpec := opctx.getServiceTemplateSpec()
	if serviceTemplateSpec == nil {
		return nil, nil
	}
	service := &corev1.Service{
		ObjectMeta: serviceTemplateSpec.ObjectMeta,
		Spec:       serviceTemplateSpec.Spec,
	}
	if service.Namespace == "" {
		service.Namespace = opctx.sensorObj.Namespace
	}
	if service.Name == "" {
		service.Name = common.DefaultServiceName(opctx.sensorObj.Name)
	}
	err := controllerscommon.SetObjectMeta(opctx.sensorObj, service, opctx.sensorObj.GroupVersionKind())
	return service, err
}

// buildSensorDeployment builds the deployment specification for the sensor
func (opctx *operationContext) buildSensorDeployment() (*appv1.Deployment, error) {
	podTemplateSpec := opctx.sensorObj.Spec.Template.DeepCopy()

	replicas := int32(1)

	deployment := &appv1.Deployment{
		ObjectMeta: podTemplateSpec.ObjectMeta,
		Spec: appv1.DeploymentSpec{
			Template: *podTemplateSpec,
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podTemplateSpec.Labels,
			},
		},
	}

	if deployment.Namespace == "" {
		deployment.Namespace = opctx.sensorObj.Namespace
	}
	if deployment.Name == "" {
		deployment.Name = opctx.sensorObj.Name
	}

	opctx.setupContainersForSensorDeployment(deployment)
	err := controllerscommon.SetObjectMeta(opctx.sensorObj, deployment, opctx.sensorObj.GroupVersionKind())

	return deployment, err
}

// containers required for sensor deployment
func (opctx *operationContext) setupContainersForSensorDeployment(deployment *appv1.Deployment) {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.SensorName,
			Value: opctx.sensorObj.Name,
		},
		{
			Name:  common.SensorNamespace,
			Value: opctx.sensorObj.Namespace,
		},
		{
			Name:  common.EnvVarSensorControllerInstanceID,
			Value: opctx.controller.Config.InstanceID,
		},
	}
	for i, container := range deployment.Spec.Template.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		deployment.Spec.Template.Spec.Containers[i] = container
	}
}

// createSensorDeployment creates the sensor deployment and
func (opctx *operationContext) createSensorDeployment() error {

}
