package gateway

import (
	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type gwResourceCtx struct {
	// gw is the gateway-controller object
	gw *v1alpha1.Gateway
	// reference to the gateway-controller-controller
	controller *GatewayController

	controllerscommon.ChildResourceContext
}

// NewGatewayResourceContext returns new gwResourceCtx
func NewGatewayResourceContext(gw *v1alpha1.Gateway, controller *GatewayController) gwResourceCtx {
	return gwResourceCtx{
		gw:         gw,
		controller: controller,
		ChildResourceContext: controllerscommon.ChildResourceContext{
			SchemaGroupVersionKind:            v1alpha1.SchemaGroupVersionKind,
			LabelOwnerName:                    common.LabelGatewayName,
			LabelKeyOwnerControllerInstanceID: common.LabelKeyGatewayControllerInstanceID,
			AnnotationOwnerResourceHashName:   common.AnnotationSensorResourceSpecHashName,
			InstanceID:                        controller.Config.InstanceID,
		},
	}
}

// gatewayResourceLabelSelector returns label selector of the gateway of the context
func (grc *gwResourceCtx) gatewayResourceLabelSelector() (labels.Selector, error) {
	req, err := labels.NewRequirement(common.LabelGatewayName, selection.Equals, []string{grc.gw.Name})
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*req), nil
}

// createGatewayService creates a given service
func (grc *gwResourceCtx) createGatewayService(svc *corev1.Service) (*corev1.Service, error) {
	return grc.controller.kubeClientset.CoreV1().Services(grc.gw.Namespace).Create(svc)
}

// deleteGatewayService deletes a given service
func (grc *gwResourceCtx) deleteGatewayService(svc *corev1.Service) error {
	return grc.controller.kubeClientset.CoreV1().Services(grc.gw.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
}

// getGatewayService returns the service of gateway
func (grc *gwResourceCtx) getGatewayService() (*corev1.Service, error) {
	selector, err := grc.gatewayResourceLabelSelector()
	if err != nil {
		return nil, err
	}
	svcs, err := grc.controller.svcInformer.Lister().Services(grc.gw.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(svcs) == 0 {
		return nil, nil
	}
	return svcs[0], nil
}

// newGatewayService returns a new service that exposes gateway.
func (grc *gwResourceCtx) newGatewayService() (*corev1.Service, error) {
	service := grc.gw.Spec.ServiceSpec.DeepCopy()
	err := grc.SetObjectMeta(grc.gw, service)
	return service, err
}

// getGatewayPod returns the pod of gateway
func (grc *gwResourceCtx) getGatewayPod() (*corev1.Pod, error) {
	selector, err := grc.gatewayResourceLabelSelector()
	if err != nil {
		return nil, err
	}
	pods, err := grc.controller.podInformer.Lister().Pods(grc.gw.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, nil
	}
	return pods[0], nil
}

// createGatewayPod creates a given pod
func (grc *gwResourceCtx) createGatewayPod(pod *corev1.Pod) (*corev1.Pod, error) {
	return grc.controller.kubeClientset.CoreV1().Pods(grc.gw.Namespace).Create(pod)
}

// deleteGatewayPod deletes a given pod
func (grc *gwResourceCtx) deleteGatewayPod(pod *corev1.Pod) error {
	return grc.controller.kubeClientset.CoreV1().Pods(grc.gw.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
}

// newGatewayPod returns a new pod of gateway
func (grc *gwResourceCtx) newGatewayPod() (*corev1.Pod, error) {
	pod := grc.gw.Spec.DeploySpec.DeepCopy()
	pod.Spec.Containers = *grc.getContainersForGatewayPod()
	err := grc.SetObjectMeta(grc.gw, pod)
	return pod, err
}

// containers required for gateway deployment
func (grc *gwResourceCtx) getContainersForGatewayPod() *[]corev1.Container {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.EnvVarGatewayNamespace,
			Value: grc.gw.Namespace,
		},
		{
			Name:  common.EnvVarGatewayEventSourceConfigMap,
			Value: grc.gw.Spec.ConfigMap,
		},
		{
			Name:  common.EnvVarGatewayName,
			Value: grc.gw.Name,
		},
		{
			Name:  common.EnvVarGatewayControllerInstanceID,
			Value: grc.controller.Config.InstanceID,
		},
		{
			Name:  common.EnvVarGatewayControllerName,
			Value: common.DefaultGatewayControllerDeploymentName,
		},
		{
			Name:  common.EnvVarGatewayServerPort,
			Value: grc.gw.Spec.ProcessorPort,
		},
	}
	containers := make([]corev1.Container, len(grc.gw.Spec.DeploySpec.Spec.Containers))
	for i, container := range grc.gw.Spec.DeploySpec.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		containers[i] = container
	}
	return &containers
}
