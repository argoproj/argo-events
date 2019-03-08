package sensor

import (
	"github.com/argoproj/argo-events/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// sensorResourceLabelSelector returns label selector of the sensor of the context
func (soc *sOperationCtx) sensorResourceLabelSelector() (labels.Selector, error) {
	req, err := labels.NewRequirement(common.LabelSensorName, selection.Equals, []string{soc.s.Name})
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*req), nil
}

// createSensorService creates a service
func (soc *sOperationCtx) createSensorService() (*corev1.Service, error) {
	svc, err := soc.newSensorService()
	if err != nil {
		return nil, err
	}
	svc, err = soc.controller.kubeClientset.CoreV1().Services(soc.s.Namespace).Create(svc)
	return svc, err
}

// getSensorService returns the service of sensor
func (soc *sOperationCtx) getSensorService() (*corev1.Service, error) {
	selector, err := soc.sensorResourceLabelSelector()
	if err != nil {
		return nil, err
	}
	svcs, err := soc.controller.svcInformer.Lister().Services(soc.s.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(svcs) == 0 {
		return nil, nil
	}
	return svcs[0], nil
}

// newSensorService returns a new service that exposes sensor.
func (soc *sOperationCtx) newSensorService() (*corev1.Service, error) {
	serviceSpec := soc.getServiceSpec()
	if serviceSpec == nil {
		return nil, nil
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.DefaultServiceName(soc.s.Name),
			Namespace: soc.s.Namespace,
		},
		Spec: *serviceSpec.DeepCopy(),
	}
	err := soc.crctx.SetObjectMeta(soc.s, service)
	return service, err
}

// getSensorPod returns the pod of sensor
func (soc *sOperationCtx) getSensorPod() (*corev1.Pod, error) {
	selector, err := soc.sensorResourceLabelSelector()
	if err != nil {
		return nil, err
	}
	pods, err := soc.controller.podInformer.Lister().Pods(soc.s.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, nil
	}
	return pods[0], nil
}

// createSensorPod creates a pod of sensor
func (soc *sOperationCtx) createSensorPod() (*corev1.Pod, error) {
	pod, err := soc.newSensorPod()
	if err != nil {
		return nil, err
	}
	pod, err = soc.controller.kubeClientset.CoreV1().Pods(soc.s.Namespace).Create(pod)
	if err != nil {
		return nil, err
	}
	return pod, nil
}

// newSensorPod returns a new pod of sensor
func (soc *sOperationCtx) newSensorPod() (*corev1.Pod, error) {
	podSpec := soc.s.Spec.DeploySpec.DeepCopy()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      soc.s.Name,
			Namespace: soc.s.Namespace,
		},
		Spec: *podSpec,
	}
	pod.Spec.Containers = *soc.getContainersForSensorPod()
	err := soc.crctx.SetObjectMeta(soc.s, pod)
	return pod, err
}

// containers required for sensor deployment
func (soc *sOperationCtx) getContainersForSensorPod() *[]corev1.Container {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.SensorName,
			Value: soc.s.Name,
		},
		{
			Name:  common.SensorNamespace,
			Value: soc.s.Namespace,
		},
		{
			Name:  common.EnvVarSensorControllerInstanceID,
			Value: soc.controller.Config.InstanceID,
		},
	}
	containers := make([]corev1.Container, len(soc.s.Spec.DeploySpec.Containers))
	for i, container := range soc.s.Spec.DeploySpec.Containers {
		container.Env = append(container.Env, envVars...)
		containers[i] = container
	}
	return &containers
}
