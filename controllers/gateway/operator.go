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
	"github.com/ghodss/yaml"
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

// operate checks the status of gateway resource and takes action based on it.
func (goc *gwOperationCtx) operate() error {
	// persist updates to gateway resource once we are done operating on it.
	defer goc.persistUpdates()
	goc.log.Info().Msg("operating on the gateway...")

	// performs a basic validation on gateway resource.
	err := Validate(goc.gw)
	if err != nil {
		goc.log.Error().Err(err).Msg("gateway validation failed")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, "validation failed")
		return err
	}

	// check the state of a gateway and take actions accordingly
	switch goc.gw.Status.Phase {
	case v1alpha1.NodePhaseNew:
		// check if the dispatch mechanism of the gateway is http
		// if it is, then we need to list of watchers because http post query requires endpoints to query to.
		var sensorWatchers []string
		var gatewayWatchers []string
		// list sensor and gateway watchers and passes this list to gateway transformer using a k8 configmap
		// gateway transformer then set a watcher for this configmap so the watcher list in configmap can be updated
		// and gateway transformer will pick up any new changes to the watchers
		if goc.gw.Spec.DispatchMechanism == v1alpha1.HTTPGateway {
			if goc.gw.Spec.Watchers.Sensors != nil {
				for _, sensor := range goc.gw.Spec.Watchers.Sensors {
					b, err := yaml.Marshal(sensor)
					if err != nil {
						goc.log.Error().Str("sensor-watcher", sensor.Name).Err(err).Msg("failed to parse sensor watcher")
						goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to parse sensor watcher. Sensor Watcher: %s Err: %+v", sensor.Name, err))
						return err
					}
					sensorWatchers = append(sensorWatchers, string(b))
				}
			}
			if goc.gw.Spec.Watchers.Gateways != nil {
				for _, gateway := range goc.gw.Spec.Watchers.Gateways {
					b, err := yaml.Marshal(gateway)
					if err != nil {
						goc.log.Error().Str("gateway-watcher", gateway.Name).Err(err).Msg("failed to parse gateway watcher")
						goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to parse gateway watcher. gateway Watcher: %s Err: %+v", gateway.Name, err))
						return err
					}
					gatewayWatchers = append(gatewayWatchers, string(b))
				}
			}
		}

		// create a configuration map for gateway transformer
		// this configmap contains information required to convert event received
		// from gateway processor into cloudevents specification compliant event
		// and the list of watchers to dispatch the event to.
		// In future, we will add different flavors of gateway transformer depending on event dispatch mechanism
		// e.g. for kafka and nats dispatch mechanism we won't need list of watchers.
		gatewayConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.DefaultGatewayTransformerConfigMapName(goc.gw.Name),
				Namespace: goc.gw.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(goc.gw, v1alpha1.SchemaGroupVersionKind),
				},
			},
			Data: map[string]string{
				// source of the event is the gateway
				common.EventSource: goc.gw.Name,
				// version of event
				common.EventTypeVersion: goc.gw.Spec.Version,
				// type of the event is type of the gateway e.g. webhook, calendar, stream etc
				common.EventType: string(goc.gw.Spec.Type),
				// list of sensor watchers
				common.SensorWatchers: strings.Join(sensorWatchers, ","),
				// list of gateway watchers
				common.GatewayWatchers: strings.Join(gatewayWatchers, ","),
			},
		}

		// create the gateway transformer configmap
		_, err = goc.controller.kubeClientset.CoreV1().ConfigMaps(goc.gw.Namespace).Create(gatewayConfigMap)
		if err != nil {
			// the error will be escalated by creating k8 event.
			goc.log.Error().Err(err).Msg("failed to create transformer gateway configuration")
			// mark gateway as failed
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create transformer gateway configuration. err: %s", err))
			return err
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
			return err
		}

		goc.log.Info().Str("deployment-name", common.DefaultGatewayDeploymentName(goc.gw.Name)).Msg("gateway deployment created")
		// expose gateway if service is configured
		if goc.gw.Spec.ServiceSpec != nil {
			svc, err := goc.createGatewayService()
			// Failed to expose gateway through a service.
			if err != nil {
				goc.log.Error().Err(err).Msg("failed to create service for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseServiceError, fmt.Sprintf("failed to create service. err: %s", err.Error()))
				return err
			}
			goc.log.Info().Str("svc-name", svc.ObjectMeta.Name).Msg("gateway service is created")
		}
		goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")

		// Gateway is in error
	case v1alpha1.NodePhaseError:
		// todo: maybe we don't need to perform any deployment update. If the gateway is in error state
		// then user will just delete the gateway and create a new one.
		gDeployment, err := goc.controller.kubeClientset.AppsV1().Deployments(goc.gw.Namespace).Get(common.DefaultGatewayDeploymentName(goc.gw.Name), metav1.GetOptions{})
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

// Creates a service that exposes gateway to other cluster components or to outside world
func (goc *gwOperationCtx) createGatewayService() (*corev1.Service, error) {
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
	// todo: this might not be required if in validation we check that if serviceSpec is
	// specified then make sure selectors are specified as well.
	if gatewayService.Spec.Selector == nil {
		gatewayService.Spec.Selector = goc.gw.ObjectMeta.Labels
	}

	svc, err := goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Create(gatewayService)
	return svc, err
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
func (goc *gwOperationCtx) markGatewayPhase(phase v1alpha1.NodePhase, message string) {
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
	goc.log.Info().Str("old-message", string(goc.gw.Status.Message)).Str("new-message", message)
	goc.gw.Status.Message = message
	goc.updated = true
}

// creates a list of containers required for gateway deployment
func (goc *gwOperationCtx) getContainersForGatewayPod() *[]corev1.Container {
	// env variables for gateway processor
	// these env vars are common to different flavors of gateway.
	envVars := []corev1.EnvVar{
		{
			Name:  common.GatewayTransformerPortEnvVar,
			Value: common.GatewayTransformerPort,
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
			Name:  common.GatewayControllerInstanceIDEnvVar,
			Value: goc.controller.Config.InstanceID,
		},
		{
			Name:  common.GatewayControllerNameEnvVar,
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

		// gateway processor server containers. Only one of these need to act as gRPC server and accept
		// connections from gateway processor client.
		eventGeneratorContainers := goc.gw.Spec.DeploySpec.Containers

		// gateway processor client container
		eventProcessorContainer := corev1.Container{
			Name:            gatewayProcessor,
			Image:           common.GatewayProcessorGRPCClientImage,
			ImagePullPolicy: corev1.PullAlways,
		}

		// this env var is available across all gateway processor server and client containers
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
		// in case of http gateway flavor, gateway processor server listens for new/stale configs on predefined API endpoints
		// similarly, gateway processor client listens to events from gateway processor server on a predefined API endpoint

		// gateway processor server containers. Only one of these needs to accept new/old configs from
		// gateway processor client
		eventGeneratorContainers := goc.gw.Spec.DeploySpec.Containers

		// gateway processor client container
		eventProcessorContainer := corev1.Container{
			Name:            gatewayProcessor,
			Image:           common.GatewayProcessorHTTPClientImage,
			ImagePullPolicy: corev1.PullAlways,
		}

		// these variables are available to all containers, both gateway processor server
		// and gateway processor client
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

		// only one of these container must have a main entry. others can act as a supporting components
		eventGeneratorAndProcessorContainers := goc.gw.Spec.DeploySpec.Containers
		containers = append(containers, eventGeneratorAndProcessorContainers...)
		for i, container := range containers {
			containers[i].Env = append(container.Env, envVars...)
		}
	}

	// Gateway transformer container. Currently only HTTP protocol is supported
	// for event dispatch mechanism. In future, we plan to add different dispatch mechanisms like
	// NATS, Kafka etc.
	var transformerImage string

	switch goc.gw.Spec.DispatchMechanism {
	case v1alpha1.HTTPGateway:
		transformerImage = common.GatewayHTTPEventTransformerImage
	case v1alpha1.KafkaGateway:
		transformerImage = common.GatewayKafkaEventTransformerImage
	case v1alpha1.NATSGateway:
		transformerImage = common.GatewayNATSEventTransformerImage
	default:
		transformerImage = common.GatewayHTTPEventTransformerImage
	}

	// create container for gateway transformer
	gatewayTransformerContainer := corev1.Container{
		Name:            gatewayTransformer,
		ImagePullPolicy: corev1.PullAlways,
		Image:           transformerImage,
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
