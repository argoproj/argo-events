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
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/pkg/errors"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getServiceSpec returns a K8s service spec for the sensor
func (opctx *operationContext) getServiceSpec() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				common.LabelSensorName:    opctx.sensor.Name,
				LabelControllerInstanceID: opctx.controller.Config.InstanceID,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       intstr.Parse(opctx.sensor.Spec.EventProtocol.Http.Port).IntVal,
					TargetPort: intstr.FromInt(int(intstr.Parse(opctx.sensor.Spec.EventProtocol.Http.Port).IntVal)),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				common.LabelSensorName:    opctx.sensor.Name,
				LabelControllerInstanceID: opctx.controller.Config.InstanceID,
			},
		},
	}
}

// buildService returns a new service that exposes sensor.
func (opctx *operationContext) buildService() (*corev1.Service, error) {
	service := opctx.getServiceSpec()
	if err := controllerscommon.SetObjectMeta(opctx.sensor, service, opctx.sensor.GroupVersionKind()); err != nil {
		return nil, err
	}
	return service, nil
}

// buildSensorDeployment builds the deployment specification for the sensor
func (opctx *operationContext) buildSensorDeployment() (*appv1.Deployment, error) {
	podTemplateSpec := opctx.sensor.Spec.Template.DeepCopy()
	replicas := int32(1)

	if podTemplateSpec.Labels == nil {
		podTemplateSpec.Labels = map[string]string{}
	}

	podTemplateSpec.Labels[common.LabelOwnerName] = opctx.sensor.Name

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

	opctx.setupContainersForDeployment(deployment)

	if err := controllerscommon.SetObjectMeta(opctx.sensor, deployment, opctx.sensor.GroupVersionKind()); err != nil {
		return nil, err
	}
	return deployment, nil
}

// containers required for sensor deployment
func (opctx *operationContext) setupContainersForDeployment(deployment *appv1.Deployment) {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.SensorName,
			Value: opctx.sensor.Name,
		},
		{
			Name:  common.SensorNamespace,
			Value: opctx.sensor.Namespace,
		},
		{
			Name:  common.EnvVarControllerInstanceID,
			Value: opctx.controller.Config.InstanceID,
		},
	}
	for i, container := range deployment.Spec.Template.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		deployment.Spec.Template.Spec.Containers[i] = container
	}
}

// createResources creates the associated K8s resources with the sensor object
func (opctx *operationContext) createResources() error {
	if opctx.sensor.Status.Resources == nil {
		opctx.sensor.Status.Resources = &v1alpha1.SensorResources{}
	}

	opctx.logger.Infoln("creating a deployment for the sensor")
	deployment, err := opctx.createDeployment()
	if err != nil {
		return err
	}
	opctx.sensor.Status.Resources.Deployment = &deployment.ObjectMeta
	opctx.logger.WithField("name", deployment.Name).WithField("namespace", deployment.Namespace).Infoln("deployment successfully created for the sensor")

	if opctx.sensor.Spec.EventProtocol.Type == apicommon.HTTP {
		opctx.logger.Infoln("creating a service for the sensor")
		service, err := opctx.createService()
		if err != nil {
			return err
		}
		opctx.sensor.Status.Resources.Service = &service.ObjectMeta
		opctx.logger.WithField("name", service.Name).WithField("namespace", service.Namespace).Infoln("service successfully created for the sensor")
	}

	return nil
}

// createDeployment creates a deployment for the sensor
func (opctx *operationContext) createDeployment() (*appv1.Deployment, error) {
	deploymentObj, err := opctx.buildSensorDeployment()
	if err != nil {
		return nil, err
	}
	return opctx.controller.k8sClient.AppsV1().Deployments(opctx.sensor.Namespace).Create(deploymentObj)
}

// createService creates a service for the sensor
func (opctx *operationContext) createService() (*corev1.Service, error) {
	serviceObj, err := opctx.buildService()
	if err != nil {
		return nil, err
	}
	return opctx.controller.k8sClient.CoreV1().Services(serviceObj.Namespace).Create(serviceObj)
}

// updateResources updates the K8s resources associated with the sensor object
func (opctx *operationContext) updateResources() error {
	deployment, err := opctx.updateDeployment()
	if err != nil {
		return err
	}
	opctx.sensor.Status.Resources.Deployment = &deployment.ObjectMeta
	service, err := opctx.updateService()
	if err != nil {
		return err
	}
	if service == nil {
		return nil
	}
	opctx.sensor.Status.Resources.Service = &service.ObjectMeta
	return nil
}

// updateDeployment updates the deployment for the sensor
func (opctx *operationContext) updateDeployment() (*appv1.Deployment, error) {
	newDeployment, err := opctx.buildSensorDeployment()
	if err != nil {
		return nil, err
	}
	oldDeploymentMetadata := opctx.sensor.Status.Resources.Deployment
	if oldDeploymentMetadata == nil {
		return nil, errors.New("deployment metadata is expected to be set in gateway object")
	}
	oldDeployment, err := opctx.controller.k8sClient.AppsV1().Deployments(oldDeploymentMetadata.Namespace).Get(oldDeploymentMetadata.Name, metav1.GetOptions{})
	if err != nil {
		if apierror.IsNotFound(err) {
			return opctx.controller.k8sClient.AppsV1().Deployments(newDeployment.Namespace).Create(newDeployment)
		}
		return nil, err
	}
	deploymentHash, err := common.GetObjectHash(newDeployment)
	if err != nil {
		return nil, err
	}
	if oldDeployment.Annotations != nil && oldDeployment.Annotations[common.AnnotationResourceSpecHash] != deploymentHash {
		if err := opctx.controller.k8sClient.AppsV1().Deployments(oldDeployment.Namespace).Delete(oldDeployment.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		return opctx.controller.k8sClient.AppsV1().Deployments(newDeployment.Namespace).Create(newDeployment)
	}
	return nil, nil
}

// updateService updates the service for the sensor
func (opctx *operationContext) updateService() (*corev1.Service, error) {
	isHttpTransport := opctx.sensor.Spec.EventProtocol.Type == apicommon.HTTP
	oldServiceMetadata := opctx.sensor.Status.Resources.Service

	if oldServiceMetadata == nil && !isHttpTransport {
		return nil, nil
	}
	if oldServiceMetadata != nil && !isHttpTransport {
		if err := opctx.controller.k8sClient.CoreV1().Services(oldServiceMetadata.Namespace).Delete(oldServiceMetadata.Name, &metav1.DeleteOptions{}); err != nil {
			// warning is sufficient instead of halting the entire sensor operation by marking it as failed.
			opctx.logger.WithField("service-name", oldServiceMetadata.Name).WithError(err).Warnln("failed to delete the service")
		}
		return nil, nil
	}
	service, err := opctx.buildService()
	if err != nil {
		return nil, err
	}
	if oldServiceMetadata == nil && isHttpTransport {
		return opctx.controller.k8sClient.CoreV1().Services(service.Namespace).Create(service)
	}
	if oldServiceMetadata == nil {
		return nil, errors.New("service metadata is expected to be set in sensor object")
	}
	serviceHash, err := common.GetObjectHash(service)
	if oldServiceMetadata.Annotations != nil && oldServiceMetadata.Annotations[common.AnnotationResourceSpecHash] != serviceHash {
		if err := opctx.controller.k8sClient.CoreV1().Services(oldServiceMetadata.Namespace).Delete(oldServiceMetadata.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		return opctx.controller.k8sClient.CoreV1().Services(service.Namespace).Create(service)
	}
}
