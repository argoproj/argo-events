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
func (src *sResourceCtx) createSensorService(svc *corev1.Service) (*corev1.Service, error) {
	return src.controller.kubeClientset.CoreV1().Services(src.s.Namespace).Create(svc)
}

// deleteSensorService deletes a given service
func (src *sResourceCtx) deleteSensorService(svc *corev1.Service) error {
	return src.controller.kubeClientset.CoreV1().Services(src.s.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
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
	serviceTemplateSpec := src.getServiceTemplateSpec()
	if serviceTemplateSpec == nil {
		return nil, nil
	}
	service := &corev1.Service{
		ObjectMeta: serviceTemplateSpec.ObjectMeta,
		Spec:       serviceTemplateSpec.Spec,
	}
	if service.Namespace == "" {
		service.Namespace = src.s.Namespace
	}
	if service.Name == "" {
		service.Name = common.DefaultServiceName(src.s.Name)
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
func (src *sResourceCtx) createSensorPod(pod *corev1.Pod) (*corev1.Pod, error) {
	return src.controller.kubeClientset.CoreV1().Pods(src.s.Namespace).Create(pod)
}

// deleteSensorPod deletes a given pod
func (src *sResourceCtx) deleteSensorPod(pod *corev1.Pod) error {
	return src.controller.kubeClientset.CoreV1().Pods(src.s.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
}

// newSensorPod returns a new pod of sensor
func (src *sResourceCtx) newSensorPod() (*corev1.Pod, error) {
	podTemplateSpec := src.s.Spec.Template.DeepCopy()
	pod := &corev1.Pod{
		ObjectMeta: podTemplateSpec.ObjectMeta,
		Spec:       podTemplateSpec.Spec,
	}
	if pod.Namespace == "" {
		pod.Namespace = src.s.Namespace
	}
	if pod.Name == "" {
		pod.Name = src.s.Name
	}
	src.setupContainersForSensorPod(pod)
	err := src.SetObjectMeta(src.s, pod)
	return pod, err
}

// containers required for sensor deployment
func (src *sResourceCtx) setupContainersForSensorPod(pod *corev1.Pod) {
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
	for i, container := range pod.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		pod.Spec.Containers[i] = container
	}
}

func (src *sResourceCtx) getServiceTemplateSpec() *pc.ServiceTemplateSpec {
	var serviceSpec *pc.ServiceTemplateSpec
	// Create a ClusterIP service to expose sensor in cluster if the event protocol type is HTTP
	if src.s.Spec.EventProtocol.Type == pc.HTTP {
		serviceSpec = &pc.ServiceTemplateSpec{
			Spec: corev1.ServiceSpec{
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
			},
		}
	}
	return serviceSpec
}
