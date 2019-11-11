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
	req, err := labels.NewRequirement(common.LabelOwnerName, selection.Equals, []string{opctx.sensorObj.Name})
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

	if podTemplateSpec.Labels == nil {
		podTemplateSpec.Labels = map[string]string{}
	}

	podTemplateSpec.Labels[common.LabelOwnerName] = opctx.sensorObj.Name

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

// installSensorResources installs the associated K8s resources with the sensor object
func (opctx *operationContext) installSensorResources() error {
	deploymentObj, err := opctx.buildSensorDeployment()
	if err != nil {
		return err
	}

	opctx.logger.Infoln("installing deployment for the sensor...")

	if _, err := opctx.controller.k8sClient.AppsV1().Deployments(opctx.sensorObj.Namespace).Create(deploymentObj); err != nil {
		return err
	}

	opctx.logger.Infoln("deployment for the sensor installed successfully")

	opctx.logger.Infoln("checking if a service is required to be deployed for the sensor...")

	serviceObj, err := opctx.buildSensorService()
	if err != nil {
		return err
	}

	if serviceObj != nil {
		opctx.logger.Infoln("installing the service for the sensor...")

		if _, err := opctx.controller.k8sClient.CoreV1().Services(opctx.sensorObj.Namespace).Create(serviceObj); err != nil {
			return err
		}

		opctx.logger.Infoln("service for the sensor is installed successfully")
		return nil
	}

	opctx.logger.Infoln("a service is not required for the sensor")
	return nil
}

// updateSensorResources updates the K8s resources associated with the sensor object
func (opctx *operationContext) updateSensorResources() error {
	opctx.logger.Infoln("fetching the label selector to list the deployment for the sensor")
	selectorLabel, err := opctx.sensorResourceLabelSelector()
	if err != nil {
		return err
	}

	deployments, err := opctx.controller.k8sClient.AppsV1().Deployments(opctx.sensorObj.Namespace).List(metav1.ListOptions{
		LabelSelector: selectorLabel.String(),
	})
	if err != nil {
		return err
	}

	deployment := deployments.Items[0]

	newDeploymentObjHash, err := common.GetObjectHash(opctx.sensorObj)
	if err != nil {
		return err
	}

	if deployment.Annotations[common.AnnotationResourceSpecHash] != newDeploymentObjHash {
		opctx.logger.Infoln("updating deployment for the sensor...")

		updatedDeploymentObj, err := opctx.buildSensorDeployment()
		if err != nil {
			return err
		}

		if _, err := opctx.controller.k8sClient.AppsV1().Deployments(opctx.sensorObj.Namespace).Update(updatedDeploymentObj); err != nil {
			return err
		}
	}

	return nil
}
