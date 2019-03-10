package sensor

import (
	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type sResourceCtx struct {
	// s is the gateway-controller object
	s *v1alpha1.Sensor
	// reference to the gateway-controller-controller
	controller *SensorController

	controllerscommon.ChildResourceContext
}

// NewSensorResourceContext returns new sResourceCtx
func NewSensorResourceContext(s *v1alpha1.Sensor, controller *SensorController) sResourceCtx {
	return sResourceCtx{
		s:          s,
		controller: controller,
		ChildResourceContext: controllerscommon.ChildResourceContext{
			SchemaGroupVersionKind:            v1alpha1.SchemaGroupVersionKind,
			LabelOwnerName:                    common.LabelSensorName,
			LabelKeyOwnerControllerInstanceID: common.LabelKeySensorControllerInstanceID,
			AnnotationOwnerResourceHashName:   common.AnnotationSensorResourceSpecHashName,
			InstanceID:                        controller.Config.InstanceID,
		},
	}
}

// sensorResourceLabelSelector returns label selector of the sensor of the context
func (src *sResourceCtx) sensorResourceLabelSelector() (labels.Selector, error) {
	req, err := labels.NewRequirement(common.LabelSensorName, selection.Equals, []string{src.s.Name})
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*req), nil
}

// createSensorService creates a service
func (src *sResourceCtx) createSensorService() (*corev1.Service, error) {
	svc, err := src.newSensorService()
	if err != nil {
		return nil, err
	}
	svc, err = src.controller.kubeClientset.CoreV1().Services(src.s.Namespace).Create(svc)
	return svc, err
}

// getSensorService returns the service of sensor
func (src *sResourceCtx) getSensorService() (*corev1.Service, error) {
	selector, err := src.sensorResourceLabelSelector()
	if err != nil {
		return nil, err
	}
	svcs, err := src.controller.svcInformer.Lister().Services(src.s.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(svcs) == 0 {
		return nil, nil
	}
	return svcs[0], nil
}

// newSensorService returns a new service that exposes sensor.
func (src *sResourceCtx) newSensorService() (*corev1.Service, error) {
	serviceSpec := src.getServiceSpec()
	if serviceSpec == nil {
		return nil, nil
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.DefaultServiceName(src.s.Name),
			Namespace: src.s.Namespace,
		},
		Spec: *serviceSpec.DeepCopy(),
	}
	err := src.SetObjectMeta(src.s, service)
	return service, err
}

// getSensorPod returns the pod of sensor
func (src *sResourceCtx) getSensorPod() (*corev1.Pod, error) {
	selector, err := src.sensorResourceLabelSelector()
	if err != nil {
		return nil, err
	}
	pods, err := src.controller.podInformer.Lister().Pods(src.s.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, nil
	}
	return pods[0], nil
}

// createSensorPod creates a pod of sensor
func (src *sResourceCtx) createSensorPod() (*corev1.Pod, error) {
	pod, err := src.newSensorPod()
	if err != nil {
		return nil, err
	}
	pod, err = src.controller.kubeClientset.CoreV1().Pods(src.s.Namespace).Create(pod)
	if err != nil {
		return nil, err
	}
	return pod, nil
}

// newSensorPod returns a new pod of sensor
func (src *sResourceCtx) newSensorPod() (*corev1.Pod, error) {
	podSpec := src.s.Spec.DeploySpec.DeepCopy()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      src.s.Name,
			Namespace: src.s.Namespace,
		},
		Spec: *podSpec,
	}
	pod.Spec.Containers = *src.getContainersForSensorPod()
	err := src.SetObjectMeta(src.s, pod)
	return pod, err
}

// containers required for sensor deployment
func (src *sResourceCtx) getContainersForSensorPod() *[]corev1.Container {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.SensorName,
			Value: src.s.Name,
		},
		{
			Name:  common.SensorNamespace,
			Value: src.s.Namespace,
		},
		{
			Name:  common.EnvVarSensorControllerInstanceID,
			Value: src.controller.Config.InstanceID,
		},
	}
	containers := make([]corev1.Container, len(src.s.Spec.DeploySpec.Containers))
	for i, container := range src.s.Spec.DeploySpec.Containers {
		container.Env = append(container.Env, envVars...)
		containers[i] = container
	}
	return &containers
}

func (src *sResourceCtx) getServiceSpec() *corev1.ServiceSpec {
	var serviceSpec *corev1.ServiceSpec
	// Create a ClusterIP service to expose sensor in cluster if the event protocol type is HTTP
	if src.s.Spec.EventProtocol.Type == pc.HTTP {
		serviceSpec = &corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       intstr.Parse(src.s.Spec.EventProtocol.Http.Port).IntVal,
					TargetPort: intstr.FromInt(int(intstr.Parse(src.s.Spec.EventProtocol.Http.Port).IntVal)),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				common.LabelSensorName:                    src.s.Name,
				common.LabelKeySensorControllerInstanceID: src.controller.Config.InstanceID,
			},
		}
	}
	return serviceSpec
}
