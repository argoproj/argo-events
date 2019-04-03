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
			AnnotationOwnerResourceHashName:   common.AnnotationGatewayResourceSpecHashName,
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
	servicTemplateSpec := grc.gw.Spec.Service.DeepCopy()
	if servicTemplateSpec == nil {
		return nil, nil
	}
	service := &corev1.Service{
		ObjectMeta: servicTemplateSpec.ObjectMeta,
		Spec:       servicTemplateSpec.Spec,
	}
	if service.Namespace == "" {
		service.Namespace = grc.gw.Namespace
	}
	if service.Name == "" {
		service.Name = common.DefaultServiceName(grc.gw.Name)
	}
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
	podTemplateSpec := grc.gw.Spec.Template.DeepCopy()
	pod := &corev1.Pod{
		ObjectMeta: podTemplateSpec.ObjectMeta,
		Spec:       podTemplateSpec.Spec,
	}
	if pod.Namespace == "" {
		pod.Namespace = grc.gw.Namespace
	}
	if pod.Name == "" {
		pod.Name = grc.gw.Name
	}
	grc.setupContainersForGatewayPod(pod)
	err := grc.SetObjectMeta(grc.gw, pod)
	return pod, err
}

// containers required for gateway deployment
func (grc *gwResourceCtx) setupContainersForGatewayPod(pod *corev1.Pod) {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.EnvVarGatewayNamespace,
			Value: grc.gw.Namespace,
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
	for i, container := range pod.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		pod.Spec.Containers[i] = container
	}
}
