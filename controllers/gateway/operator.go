package gateway

import (
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	client "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/typed/gateway/v1alpha1"
	zlog "github.com/rs/zerolog"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"

	"fmt"
	"github.com/argoproj/argo-events/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"
	"time"
)

const (
	gatewayProcessor   = "gateway-processor"
	gatewayTransformer = "gateway-transformer"
)

// the context of an operation on a gateway-controller.
// the gateway-controller-controller creates this context each time it picks a Gateway off its queue.
type gwOperationCtx struct {
	// gw is the gateway-controller object
	gw *v1alpha1.Gateway
	// updated indicates whether the gateway-controller object was updated and needs to be persisted back to k8
	updated bool
	// log is the logger for a gateway
	log zlog.Logger
	// reference to the gateway-controller-controller
	controller *GatewayController
}

// newGatewayOperationCtx creates and initializes a new gOperationCtx object
func newGatewayOperationCtx(gw *v1alpha1.Gateway, controller *GatewayController) *gwOperationCtx {
	return &gwOperationCtx{
		gw:         gw.DeepCopy(),
		updated:    false,
		log:        zlog.New(os.Stdout).With().Str("name", gw.Name).Str("namespace", gw.Namespace).Logger(),
		controller: controller,
	}
}

func (goc *gwOperationCtx) operate() error {
	defer goc.persistUpdates()
	goc.log.Info().Msg("operating on the gateway")

	// validate the gateway
	err := goc.validate()
	if err != nil {
		goc.log.Error().Err(err).Msg("gateway validation failed")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, "validation failed")
		return err
	}

	// manages states of a gateway
	switch goc.gw.Status.Phase {
	case v1alpha1.NodePhaseNew:
		// declare the configuration map for gateway transformer
		gatewayConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.DefaultGatewayTransformerConfigMapName(goc.gw.Name),
				Namespace: goc.gw.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(goc.gw, v1alpha1.SchemaGroupVersionKind),
				},
			},
			Data: map[string]string{
				common.EventSource:      goc.gw.Name,
				common.EventTypeVersion: goc.gw.Spec.Version,
				common.EventType:        goc.gw.Spec.Type,
				common.SensorList:       strings.Join(goc.gw.Spec.Sensors, ","),
			},
		}
		// create gateway transformer configmap
		_, err = goc.controller.kubeClientset.CoreV1().ConfigMaps(goc.gw.Namespace).Create(gatewayConfigMap)
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to create transformer gateway configuration")
			// mark gateway as failed
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create transformer gateway configuration. err: %s", err))
		}

		// add gateway name to gateway label
		goc.gw.ObjectMeta.Labels[common.LabelGatewayName] = goc.gw.Name

		// declare the gateway deployment. The deployment has two components,
		// 1) Gateway Processor   - Either generates events internally or listens to outside world events.
		//                          and dispatches the event to gateway transformer
		// 2) Gateway Transformer - Listens for events from gateway processor, convert them into cloudevents specification
		//                          compliant events and dispatches them to sensors of interest.
		gatewayDeployment := &appv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.DefaultGatewayDeploymentName(goc.gw.Name),
				Namespace: goc.gw.Namespace,
				Labels:    goc.gw.ObjectMeta.Labels,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(goc.gw, v1alpha1.SchemaGroupVersionKind),
				},
			},
			Spec: appv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: goc.gw.ObjectMeta.Labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: goc.gw.ObjectMeta.Labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: goc.gw.Spec.DeploySpec.ServiceAccountName,
						Containers:         *goc.getContainersForGatewayPod(),
					},
				},
			},
		}

		// we can now create the gateway deployment.
		// depending on user configuration gateway will be exposed outside the cluster or intra-cluster.
		_, err = goc.controller.kubeClientset.AppsV1().Deployments(goc.gw.Namespace).Create(gatewayDeployment)
		if err != nil {
			goc.log.Error().Err(err).Msg("failed gateway deployment")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed gateway deployment. err: %s", err))
		} else {
			// expose gateway if service is configured
			if goc.gw.Spec.ServiceSpec != nil {
				err = goc.createGatewayService()
				if err != nil {
					goc.log.Error().Err(err).Msg("failed to create service for gateway")
					goc.markGatewayPhase(v1alpha1.NodePhaseServiceError, "failed to create service", err.Error())
					return err
				}
			}
			goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")
		}

		// Gateway is in error
	case v1alpha1.NodePhaseError:
		gDeployment, err := goc.controller.kubeClientset.AppsV1().Deployments(goc.gw.Namespace).Get(goc.gw.Name, metav1.GetOptions{})
		if err != nil {
			goc.log.Error().Err(err).Msg("error occurred retrieving gateway deployment")
			return err
		}

		// self heal by updating container images if any
		gDeployment.Spec.Template.Spec.Containers = *goc.getContainersForGatewayPod()
		_, err = goc.controller.kubeClientset.AppsV1().Deployments(goc.gw.Namespace).Update(gDeployment)
		if err != nil {
			goc.log.Error().Err(err).Msg("error occurred updating gateway deployment")
			return err
		}

		// Update node phase to running
		goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")

		// Gateway is already running, do nothing
	case v1alpha1.NodePhaseRunning:
		goc.log.Info().Msg("gateway is running")
		// Failed to expose gateway through a service.
	case v1alpha1.NodePhaseServiceError:
		// check whether service is now created successfully. service's name must be gateway name followed by "-gateway-svc".
		_, err := goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Get(common.DefaultGatewayServiceName(goc.gw.Name), metav1.GetOptions{})
		if err != nil {
			goc.log.Warn().Str("expected-service", common.DefaultGatewayServiceName(goc.gw.Name)).Err(err).Msg("no service found")
			return err
		}
		// Update node phase to running
		goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")
	default:
		goc.log.Panic().Str("phase", string(goc.gw.Status.Phase)).Msg("unknown gateway phase.")
	}
	return nil
}

// Creates a service that exposes gateway outside the cluster
func (goc *gwOperationCtx) createGatewayService() error {
	gatewayService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.DefaultGatewayServiceName(goc.gw.Name),
			Namespace: goc.gw.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(goc.gw, v1alpha1.SchemaGroupVersionKind),
			},
		},
		Spec: *goc.gw.Spec.ServiceSpec,
	}

	// if selector is not provided, override selectors with gateway labels
	if gatewayService.Spec.Selector == nil {
		gatewayService.Spec.Selector = goc.gw.ObjectMeta.Labels
	}

	_, err := goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Create(gatewayService)
	return err
}

// persist the updates to the gateway resource
func (goc *gwOperationCtx) persistUpdates() {
	var err error
	if !goc.updated {
		return
	}
	gatewayClient := goc.controller.gatewayClientset.ArgoprojV1alpha1().Gateways(goc.gw.ObjectMeta.Namespace)
	goc.gw, err = gatewayClient.Update(goc.gw)
	if err != nil {
		goc.log.Warn().Err(err).Msg("error updating gateway")
		if errors.IsConflict(err) {
			return
		}
		goc.log.Info().Msg("re-applying updates on latest version and retrying update")
		err = goc.reapplyUpdate(gatewayClient)
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to re-apply update")
			return
		}
	}
	goc.log.Info().Str("gateway-phase", string(goc.gw.Status.Phase)).Msg("gateway state updated successfully")
	time.Sleep(1 * time.Second)
}

// reapply the updates to gateway resource
func (goc *gwOperationCtx) reapplyUpdate(gatewayClient client.GatewayInterface) error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		g, err := gatewayClient.Get(goc.gw.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		g.Status = goc.gw.Status
		goc.gw, err = gatewayClient.Update(g)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
}

// mark the overall gateway phase
func (goc *gwOperationCtx) markGatewayPhase(phase v1alpha1.NodePhase, message ...string) {
	justCompleted := goc.gw.Status.Phase != phase
	if justCompleted {
		goc.log.Info().Str("old-phase", string(goc.gw.Status.Phase)).Str("new-phase", string(phase))
		goc.gw.Status.Phase = phase
		goc.updated = true
		if goc.gw.ObjectMeta.Labels == nil {
			goc.gw.ObjectMeta.Labels = make(map[string]string)
		}
		if goc.gw.ObjectMeta.Annotations == nil {
			goc.gw.ObjectMeta.Annotations = make(map[string]string)
		}
		goc.gw.ObjectMeta.Labels[common.LabelKeyPhase] = string(phase)
		// add annotations so a resource sensor can watch this gateway.
		goc.gw.ObjectMeta.Annotations[common.GatewayLabelKeyPhase] = string(phase)
	}
	if goc.gw.Status.StartedAt.IsZero() {
		goc.gw.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
		goc.updated = true
	}
	if len(message) > 0 && goc.gw.Status.Message != message[0] {
		goc.log.Info().Str("old-message", string(goc.gw.Status.Message)).Str("new-message", message[0])
		goc.gw.Status.Message = message[0]
		goc.updated = true
	}
}

// creates a list of containers required for gateway deployment
func (goc *gwOperationCtx) getContainersForGatewayPod() *[]corev1.Container {
	// env variables for gateway processor
	envVars := []corev1.EnvVar{
		{
			Name:  common.GatewayTransformerPortEnvVar,
			Value: fmt.Sprintf("%d", common.GatewayTransformerPort),
		},
		{
			Name:  common.EnvVarNamespace,
			Value: goc.gw.Namespace,
		},
		{
			Name:  common.GatewayProcessorConfigMapEnvVar,
			Value: goc.gw.Spec.ConfigMap,
		},
		{
			Name:  common.GatewayName,
			Value: goc.gw.Name,
		},
		{
			Name: common.GatewayControllerInstanceIDEnvVar,
			Value: goc.controller.Config.InstanceID,
		},
		{
			Name: common.GatewayControllerNameEnvVar,
			Value: common.DefaultGatewayControllerDeploymentName,
		},
	}

	var containers []corev1.Container
	// check if gateway deployment is a gRPC server
	// deploySpec contains list of containers as user may choose to have multiple containers acting as event generators
	// Multiple containers can act as supporting containers helping to generate events
	if goc.gw.Spec.RPCPort != "" {
		// in case of gRPC gateway, gateway processor client(eventProcessor) will connect with container that has rpc server running
		// on RPCPort.
		eventGeneratorContainers := goc.gw.Spec.DeploySpec.Containers
		eventProcessorContainer := corev1.Container{
			Name:            gatewayProcessor,
			Image:           common.GatewayProcessorGRPCClientImage,
			ImagePullPolicy: corev1.PullAlways,
		}

		rpcGatewayEnvVars := append(envVars, corev1.EnvVar{
			Name:  common.GatewayProcessorGRPCServerPort,
			Value: goc.gw.Spec.RPCPort,
		})

		containers = append(containers, eventGeneratorContainers...)
		containers = append(containers, eventProcessorContainer)

		for i, container := range containers {
			containers[i].Env = append(container.Env, rpcGatewayEnvVars...)
		}
	} else if goc.gw.Spec.HTTPServerPort != "" {
		eventGeneratorContainers := goc.gw.Spec.DeploySpec.Containers
		eventProcessorContainer := corev1.Container{
			Name:            gatewayProcessor,
			Image:           common.GatewayProcessorHTTPClientImage,
			ImagePullPolicy: corev1.PullAlways,
		}

		httpEnvVars := []corev1.EnvVar{
			{
				Name:  common.GatewayProcessorServerHTTPPortEnvVar,
				Value: goc.gw.Spec.HTTPServerPort,
			},
			{
				Name:  common.GatewayProcessorClientHTTPPortEnvVar,
				Value: common.GatewayProcessorClientHTTPPort,
			},
			{
				Name:  common.GatewayProcessorHTTPServerConfigStartEndpointEnvVar,
				Value: common.GatewayProcessorHTTPServerConfigStartEndpoint,
			},
			{
				Name:  common.GatewayProcessorHTTPServerConfigStopEndpointEnvVar,
				Value: common.GatewayProcessorHTTPServerConfigStopEndpoint,
			},
			{
				Name:  common.GatewayProcessorHTTPServerEventEndpointEnvVar,
				Value: common.GatewayProcessorHTTPServerEventEndpoint,
			},
		}

		httpGatewayEnvVars := append(envVars, httpEnvVars...)

		containers = append(containers, eventGeneratorContainers...)
		containers = append(containers, eventProcessorContainer)

		for i, container := range containers {
			containers[i].Env = append(container.Env, httpGatewayEnvVars...)
		}
	} else {
		// this is when user is deploying gateway by either following core gateways or writing a completely custom gateway
		eventGeneratorAndProcessorContainers := goc.gw.Spec.DeploySpec.Containers
		containers = append(containers, eventGeneratorAndProcessorContainers...)
		for i, container := range containers {
			containers[i].Env = append(container.Env, envVars...)
		}
	}

	// create container for gateway transformer
	gatewayTransformerContainer := corev1.Container{
		Name:            gatewayTransformer,
		ImagePullPolicy: corev1.PullAlways,
		Image:           common.GatewayEventTransformerImage,
		Env: []corev1.EnvVar{
			{
				Name:  common.GatewayTransformerConfigMapEnvVar,
				Value: common.DefaultGatewayTransformerConfigMapName(goc.gw.Name),
			},
			{
				Name:  common.EnvVarNamespace,
				Value: goc.gw.Namespace,
			},
		},
	}

	containers = append(containers, gatewayTransformerContainer)

	return &containers
}
