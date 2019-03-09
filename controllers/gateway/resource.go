package gateway

import (
	"github.com/argoproj/argo-events/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// gatewayResourceLabelSelector returns label selector of the gateway of the context
func (goc *gwOperationCtx) gatewayResourceLabelSelector() (labels.Selector, error) {
	req, err := labels.NewRequirement(common.LabelGatewayName, selection.Equals, []string{goc.gw.Name})
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*req), nil
}

// createGatewayService creates a given service
func (goc *gwOperationCtx) createGatewayService(svc *corev1.Service) (*corev1.Service, error) {
	return goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Create(svc)
}

// deleteGatewayService deletes a given service
func (goc *gwOperationCtx) deleteGatewayService(svc *corev1.Service) error {
	return goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
}

// getGatewayService returns the service of gateway
func (goc *gwOperationCtx) getGatewayService() (*corev1.Service, error) {
	selector, err := goc.gatewayResourceLabelSelector()
	if err != nil {
		return nil, err
	}
	svcs, err := goc.controller.svcInformer.Lister().Services(goc.gw.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(svcs) == 0 {
		return nil, nil
	}
	return svcs[0], nil
}

// newGatewayService returns a new service that exposes gateway.
func (goc *gwOperationCtx) newGatewayService() (*corev1.Service, error) {
	service := goc.gw.Spec.ServiceSpec.DeepCopy()
	err := goc.crctx.SetObjectMeta(goc.gw, service)
	return service, err
}

// getGatewayPod returns the pod of gateway
func (goc *gwOperationCtx) getGatewayPod() (*corev1.Pod, error) {
	selector, err := goc.gatewayResourceLabelSelector()
	if err != nil {
		return nil, err
	}
	pods, err := goc.controller.podInformer.Lister().Pods(goc.gw.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, nil
	}
	return pods[0], nil
}

// createGatewayPod creates a given pod
func (goc *gwOperationCtx) createGatewayPod(pod *corev1.Pod) (*corev1.Pod, error) {
	return goc.controller.kubeClientset.CoreV1().Pods(goc.gw.Namespace).Create(pod)
}

// deleteGatewayPod deletes a given pod
func (goc *gwOperationCtx) deleteGatewayPod(pod *corev1.Pod) error {
	return goc.controller.kubeClientset.CoreV1().Pods(goc.gw.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
}

// newGatewayPod returns a new pod of gateway
func (goc *gwOperationCtx) newGatewayPod() (*corev1.Pod, error) {
	pod := goc.gw.Spec.DeploySpec.DeepCopy()
	pod.Spec.Containers = *goc.getContainersForGatewayPod()
	err := goc.crctx.SetObjectMeta(goc.gw, pod)
	return pod, err
}

// containers required for gateway deployment
func (goc *gwOperationCtx) getContainersForGatewayPod() *[]corev1.Container {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.EnvVarGatewayNamespace,
			Value: goc.gw.Namespace,
		},
		{
			Name:  common.EnvVarGatewayEventSourceConfigMap,
			Value: goc.gw.Spec.ConfigMap,
		},
		{
			Name:  common.EnvVarGatewayName,
			Value: goc.gw.Name,
		},
		{
			Name:  common.EnvVarGatewayControllerInstanceID,
			Value: goc.controller.Config.InstanceID,
		},
		{
			Name:  common.EnvVarGatewayControllerName,
			Value: common.DefaultGatewayControllerDeploymentName,
		},
		{
			Name:  common.EnvVarGatewayServerPort,
			Value: goc.gw.Spec.ProcessorPort,
		},
	}
	containers := make([]corev1.Container, len(goc.gw.Spec.DeploySpec.Spec.Containers))
	for i, container := range goc.gw.Spec.DeploySpec.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		containers[i] = container
	}
	return &containers
}
