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

package gateways

import (
	"bytes"
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	gwv1alpha1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwClientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"os"
	"time"
	"github.com/argoproj/argo-events/pkg/apis/gateway"
)

// GatewayConfig provides a generic configuration for a gateway
type GatewayConfig struct {
	// Log provides fast and simple logger dedicated to JSON output
	Log zerolog.Logger
	// Clientset is client for kubernetes API
	Clientset *kubernetes.Clientset
	// Name is gateway name
	Name string
	// Namespace is namespace for the gateway to run inside
	Namespace string
	// KubeConfig rest client config
	KubeConfig *rest.Config

	// gateway holds Gateway custom resource
	gw *gwv1alpha1.Gateway
	// gwClientset is gateway clientset
	gwcs gwClientset.Interface
	// transformerPort is gateway transformer port to dispatch event to
	transformerPort string
	// registeredConfigs stores information about current configurations that are running in the gateway
	registeredConfigs map[string]*ConfigContext
	// configName is name of configmap that contains run configuration/s for the gateway
	configName string
}

// ConfigData contains information of a configuration for gateway to run.
type ConfigContext struct {
	// Data holds the actual configuration
	Data *ConfigData
	// StopCh is used to send a stop signal to configuration runner/executor
	StopCh chan struct{}
	// Active tracks configuration state as running or stopped
	Active bool
	// Cancel is called to cancel the context used by client to communicate with gRPC server.
	// Use it only if gateway is implemented as gRPC server.
	Cancel context.CancelFunc
}

// ConfigData holds the actual configuration
type ConfigData struct {
	// Src contains name of the configuration
	Src string `json:"src"`
	// Config contains the configuration
	Config string `json:"config"`
}

// GatewayEvent is the internal representation of an event.
type GatewayEvent struct {
	// Src is source of event
	Src string `json:"src"`
	// Payload contains event data
	Payload []byte `json:"payload"`
}

// HTTPGatewayServerConfig contains information regarding http ports, endpoints
type HTTPGatewayServerConfig struct {
	// HTTPServerPort is the port on which gateway processor server is runnung
	HTTPServerPort string
	// HTTPClientPort is the port on which gateway processor client is running
	HTTPClientPort string
	// ConfigActivateEndpoint is REST endpoint listening for new configurations to run.
	ConfigActivateEndpoint string
	// ConfigurationDeactivateEndpoint is REST endpoint listening to deactivate active configuration
	ConfigurationDeactivateEndpoint string
	// EventEndpoint is REST endpoint on which gateway processor server sends events to gateway processor client
	EventEndpoint string
	// GwConfig holds generic gateway configuration
	GwConfig *GatewayConfig
	// Data holds the actual configuration
	Data *ConfigData
}

// newEventWatcher creates a new event watcher.
func (gc *GatewayConfig) newEventWatcher() *cache.ListWatch {
	x := gc.Clientset.CoreV1().RESTClient()
	resource := "events"
	labelSelector := fields.ParseSelectorOrDie(fmt.Sprintf("%s=%s", common.LabelGatewayName, gc.Name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.LabelSelector = labelSelector.String()
		options.Watch = true
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// updateGatewayResource updates gateway resource
func (gc *GatewayConfig) updateGatewayResource(event *corev1.Event) error {
	// get node/configuration to update
	nodeID, ok := event.ObjectMeta.Labels[common.LabelGatewayConfigurationName]
	if !ok {
		return fmt.Errorf("failed to update gateway resource. no configuration name provided")
	}
	node, ok := gc.gw.Status.Nodes[nodeID]
	// initialize the configuration
	if !ok {
		gc.Log.Warn().Str("config-name", nodeID).Msg("configuration not registered with gateway resource. initializing configuration...")
		gc.initializeNode(nodeID, "initialized")
		return gc.PersistUpdates()
	}
	gc.Log.Warn().Str("config-name", nodeID).Msg("updating gateway resource...")
	// check if this event happened after last updated time for the configuration
	if node.UpdateTime.Time.Before(event.EventTime.Time) {
		node.UpdateTime = event.EventTime
		switch gwv1alpha1.NodePhase(event.Action) {
		case gwv1alpha1.NodePhaseInitialized, gwv1alpha1.NodePhaseRunning, gwv1alpha1.NodePhaseCompleted, gwv1alpha1.NodePhaseError:
			gc.gw.Status.Nodes[nodeID] = node
			gc.MarkGatewayNodePhase(nodeID, gwv1alpha1.NodePhase(event.Action), event.Reason)
			// mark event as seen
			event.ObjectMeta.Labels[common.LabelGatewayEventSeen] = "true"
			_, err := gc.Clientset.CoreV1().Events(gc.Namespace).Update(event)
			if err != nil {
				gc.Log.Error().Err(err).Str("event-name", event.ObjectMeta.Name).Msg("failed to mark event as seen")
			}
			return gc.PersistUpdates()
		default:
			return fmt.Errorf("invalid gateway k8 event. unknown gateway phase, %s", event.Action)
		}
	} else {
		gc.Log.Warn().Str("config-name", nodeID).Str("node-last-update-time", node.UpdateTime.Time.String()).Str("event-time", event.EventTime.Time.String()).Msg("stale event. skipping update...")
		return nil
	}
}

func (gc *GatewayConfig) filterEvent(event *corev1.Event) bool {
	if event.Type == gateway.Kind && event.Source.Component == gc.Name && event.ObjectMeta.Labels[common.LabelGatewayEventSeen] == "" {
		return true
	}
	gc.Log.Debug().Str("event-name", event.ObjectMeta.Name).Msg("ignoring event")
	return false
}

// WatchGatewayEvents watches events generated in namespace
func (gc *GatewayConfig) WatchGatewayEvents(ctx context.Context) (cache.Controller, error) {
	source := gc.newEventWatcher()
	_, controller := cache.NewInformer(
		source,
		&corev1.Event{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if newEvent, ok := obj.(*corev1.Event); ok {
					if gc.filterEvent(newEvent) {
						gc.Log.Info().Msg("detected new k8 Event. Updating gateway resource.")
						err := gc.updateGatewayResource(newEvent)
						if err != nil {
							gc.Log.Error().Err(err).Msg("update of gateway resource failed")
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if event, ok := new.(*corev1.Event); ok {
					if gc.filterEvent(event) {
						gc.Log.Info().Msg("detected k8 Event update. Updating gateway resource.")
						err := gc.updateGatewayResource(event)
						if err != nil {
							gc.Log.Error().Err(err).Msg("update of gateway resource failed")
						}
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// WatchGatewayConfigMap watches change in configuration for the gateway
func (gc *GatewayConfig) WatchGatewayConfigMap(ctx context.Context, configManager func(config *ConfigContext) error, configDeactivator func(config *ConfigContext) error) (cache.Controller, error) {
	source := gc.newConfigMapWatch(gc.configName)
	_, controller := cache.NewInformer(
		source,
		&corev1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if newCm, ok := obj.(*corev1.ConfigMap); ok {
					gc.Log.Info().Str("config-map", gc.configName).Msg("detected ConfigMap addition. Updating the controller run config.")
					err := gc.manageConfigurations(configManager, configDeactivator, newCm)
					if err != nil {
						gc.Log.Error().Err(err).Msg("update of run config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if cm, ok := new.(*corev1.ConfigMap); ok {
					gc.Log.Info().Msg("detected ConfigMap update. Updating the controller run config.")
					err := gc.manageConfigurations(configManager, configDeactivator, cm)
					if err != nil {
						gc.Log.Error().Err(err).Msg("update of run config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newConfigMapWatch creates a new configmap watcher
func (gc *GatewayConfig) newConfigMapWatch(name string) *cache.ListWatch {
	x := gc.Clientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// manageConfigurations syncs registered configurations and updated gateway configmap
func (gc *GatewayConfig) manageConfigurations(configActivator func(config *ConfigContext) error, configDeactivator func(config *ConfigContext) error, cm *corev1.ConfigMap) error {
	newConfigs, err := gc.createInternalConfigs(cm)
	if err != nil {
		return err
	}
	staleConfigKeys, newConfigKeys := gc.diffConfigurations(newConfigs)
	gc.Log.Debug().Interface("stale-config", staleConfigKeys).Msg("stale configurations")
	gc.Log.Debug().Interface("new-config-keys", newConfigKeys).Msg("new configurations")

	// run new configurations
	for _, newConfigKey := range newConfigKeys {
		newConfig := newConfigs[newConfigKey]
		gc.registeredConfigs[newConfigKey] = newConfig
		// check if this is reactivation of configuration
		node := gc.getNodeByName(newConfig.Data.Src)
		if node != nil {
			// no need to initialize the node as this is a reactivation
			gc.Log.Info().Str("config-key", newConfig.Data.Src).Msg("reactivating configuration...")
		} else {
			gc.Log.Info().Str("config-key", newConfig.Data.Src).Msg("activating configuration...")
			// create k8 event for new configuration
			newConfigEvent := gc.K8Event("new configuration", gwv1alpha1.NodePhaseInitialized, newConfig.Data.Src)
			err = gc.CreateK8Event(newConfigEvent)
			if err != nil {
				gc.Log.Error().Str("config-name", newConfig.Data.Src).Err(err).Msg("failed to create k8 event to update gateway configurations. skipping configuration...")
				continue
			}
		}

		// run configuration
		go configActivator(newConfig)
	}

	// remove stale configurations
	for _, staleConfigKey := range staleConfigKeys {
		staleConfig := gc.registeredConfigs[staleConfigKey]
		err := configDeactivator(staleConfig)
		if err == nil {
			gc.Log.Info().Str("config", staleConfig.Data.Src).Msg("configuration deactivated.")
			delete(gc.registeredConfigs, staleConfigKey)
		} else {
			gc.Log.Error().Str("config", staleConfig.Data.Src).Err(err).Msg("failed to deactivate the configuration.")
		}
	}
	return nil
}

// DispatchEvent dispatches event to gateway transformer for further processing
func (gc *GatewayConfig) DispatchEvent(gatewayEvent *GatewayEvent) error {
	payload, err := TransformerPayload(gatewayEvent.Payload, gatewayEvent.Src)
	if err != nil {
		gc.Log.Warn().Str("config-key", gatewayEvent.Src).Err(err).Msg("failed to transform request body.")
		return err
	} else {
		gc.Log.Info().Str("config-key", gatewayEvent.Src).Msg("dispatching the event to gateway-transformer...")

		_, err = http.Post(fmt.Sprintf("http://localhost:%s", gc.transformerPort), "application/octet-stream", bytes.NewReader(payload))
		if err != nil {
			gc.Log.Warn().Str("config-key", gatewayEvent.Src).Err(err).Msg("failed to dispatch event to gateway-transformer.")
			return err
		}
	}
	return nil
}

// createInternalConfigs creates an internal representation of configuration declared in the gateway configmap.
// returned configurations are map of hash of configuration and configuration itself.
// Creating a hash of configuration makes it easy to check equality of two configurations.
func (gc *GatewayConfig) createInternalConfigs(cm *corev1.ConfigMap) (map[string]*ConfigContext, error) {
	configs := make(map[string]*ConfigContext)
	for configKey, configValue := range cm.Data {
		hashKey := Hasher(configValue)
		gc.Log.Info().Str("config-key", configKey).Interface("config-data", configValue).Str("hash", string(hashKey)).Msg("configuration hash")

		configs[hashKey] = &ConfigContext{
			Data: &ConfigData{
				Src:    configKey,
				Config: configValue,
			},
			StopCh: make(chan struct{}),
		}
	}
	return configs, nil
}

// diffConfig diffs currently registered configurations and the configurations in the gateway configmap
// It simply matches the configuration strings. So, if configuration string differs through some sequence of definition
// and although the configurations are actually same, this method will treat them as different configurations.
// retunrs staleConfig - configurations to be removed from gateway
// newConfig - new configurations to run
func (gc *GatewayConfig) diffConfigurations(newConfigs map[string]*ConfigContext) (staleConfigKeys []string, newConfigKeys []string) {
	var currentConfigKeys []string
	var updatedConfigKeys []string

	for currentConfigKey, _ := range gc.registeredConfigs {
		currentConfigKeys = append(currentConfigKeys, currentConfigKey)
	}
	for updatedConfigKey, _ := range newConfigs {
		updatedConfigKeys = append(updatedConfigKeys, updatedConfigKey)
	}

	gc.Log.Debug().Interface("current-config-keys", currentConfigKeys).Msg("hashes")
	gc.Log.Debug().Interface("updated-config-keys", updatedConfigKeys).Msg("hashes")

	swapped := false
	// iterates over current configurations and updated configurations
	// and creates two arrays, first one containing configurations that need to removed
	// and second containing new configurations that need to be added and run.
	for i := 0; i < 2; i++ {
		for _, cc := range currentConfigKeys {
			found := false
			for _, uc := range updatedConfigKeys {
				if cc == uc {
					found = true
					break
				}
			}
			if !found {
				if swapped {
					newConfigKeys = append(newConfigKeys, cc)
				} else {
					staleConfigKeys = append(staleConfigKeys, cc)
				}
			}
		}
		if i == 0 {
			currentConfigKeys, updatedConfigKeys = updatedConfigKeys, currentConfigKeys
			swapped = true
		}
	}
	return
}

// NewGatewayConfiguration returns a new gateway configuration
func NewGatewayConfiguration() *GatewayConfig {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	name, ok := os.LookupEnv(common.GatewayName)
	if !ok {
		panic("gateway name not provided")
	}
	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		panic("no namespace provided")
	}
	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}
	configName, ok := os.LookupEnv(common.GatewayProcessorConfigMapEnvVar)
	if !ok {
		panic("gateway processor configmap is not provided")
	}
	log := zlog.New(os.Stdout).With().Str("gateway-name", name).Logger()
	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gwcs := gwClientset.NewForConfigOrDie(restConfig)
	gw, err := gwcs.ArgoprojV1alpha1().Gateways(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Panic().Str("gateway-name", name).Err(err).Msg("failed to get gateway resource")
	}
	return &GatewayConfig{
		Log:               log,
		Clientset:         clientset,
		Namespace:         namespace,
		Name:              name,
		KubeConfig:        restConfig,
		registeredConfigs: make(map[string]*ConfigContext),
		transformerPort:   transformerPort,
		configName:        configName,
		gwcs:              gwcs,
		gw:                gw,
	}
}

// NewHTTPGatewayServerConfig returns a new HTTPGatewayServerConfig
func NewHTTPGatewayServerConfig() *HTTPGatewayServerConfig {
	httpGatewayServerConfig := &HTTPGatewayServerConfig{}
	httpGatewayServerConfig.HTTPServerPort = func() string {
		httpServerPort, ok := os.LookupEnv(common.GatewayProcessorServerHTTPPortEnvVar)
		if !ok {
			panic("gateway server http port is not provided")
		}
		return httpServerPort
	}()
	httpGatewayServerConfig.HTTPClientPort = func() string {
		httpClientPort, ok := os.LookupEnv(common.GatewayProcessorClientHTTPPortEnvVar)
		if !ok {
			panic("gateway client http port is not provided")
		}
		return httpClientPort
	}()
	httpGatewayServerConfig.ConfigActivateEndpoint = func() string {
		configActivateEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerConfigStartEndpointEnvVar)
		if !ok {
			panic("gateway config activation endpoint is not provided")
		}
		return configActivateEndpoint
	}()
	httpGatewayServerConfig.ConfigurationDeactivateEndpoint = func() string {
		configDeactivateEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerConfigStopEndpointEnvVar)
		if !ok {
			panic("gateway config deactivation endpoint is not provided")
		}
		return configDeactivateEndpoint
	}()
	httpGatewayServerConfig.EventEndpoint = func() string {
		eventEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerEventEndpointEnvVar)
		if !ok {
			panic("gateway event endpoint is not provided")
		}
		return eventEndpoint
	}()
	httpGatewayServerConfig.GwConfig = NewGatewayConfiguration()
	return httpGatewayServerConfig
}

// create a new node
func (gc *GatewayConfig) initializeNode(nodeName string, messages string) gwv1alpha1.NodeStatus {
	if gc.gw.Status.Nodes == nil {
		gc.gw.Status.Nodes = make(map[string]gwv1alpha1.NodeStatus)
	}
	nodeID := nodeName
	gc.Log.Info().Str("node-id", nodeID).Msg("node")
	oldNode, ok := gc.gw.Status.Nodes[nodeID]
	if ok {
		gc.Log.Info().Str("node-name", nodeName).Msg("node already initialized")
		return oldNode
	}
	node := gwv1alpha1.NodeStatus{
		ID:          nodeID,
		Name:        nodeName,
		DisplayName: nodeName,
		Phase:       gwv1alpha1.NodePhaseInitialized,
		StartedAt:   metav1.MicroTime{Time: time.Now().UTC()},
	}
	node.Message = messages
	gc.gw.Status.Nodes[nodeID] = node
	gc.Log.Info().Str("node-name", node.DisplayName).Str("node-message", node.Message).Msg("node is initialized")
	return node
}

// getNodeByName returns the node from this gateway for the nodeName
func (gc *GatewayConfig) getNodeByName(nodeName string) *gwv1alpha1.NodeStatus {
	node, ok := gc.gw.Status.Nodes[nodeName]
	if !ok {
		return nil
	}
	return &node
}

// markNodePhase marks the node with a phase, returns the node
func (gc *GatewayConfig) MarkGatewayNodePhase(nodeName string, phase gwv1alpha1.NodePhase, message string) *gwv1alpha1.NodeStatus {
	gc.Log.Debug().Str("node-name", nodeName).Msg("marking node phase...")
	gc.Log.Info().Interface("nodes", gc.gw.Status.Nodes).Msg("nodes")
	node := gc.getNodeByName(nodeName)
	if node == nil {
		gc.Log.Warn().Str("node-name", nodeName).Msg("node is not initialized")
		return nil
	}
	if node.Phase != gwv1alpha1.NodePhaseCompleted && node.Phase != phase {
		gc.Log.Info().Str("node-name", node.Name).Str("phase", string(node.Phase)).Msg("phase marked")
		node.Phase = phase
	}
	node.Message = message
	gc.gw.Status.Nodes[node.ID] = *node
	return node
}

// persist the updates to the Gateway resource
func (gc *GatewayConfig) PersistUpdates() error {
	var err error
	gc.gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gc.Namespace).Update(gc.gw)
	if err != nil {
		gc.Log.Warn().Err(err).Msg("error updating gateway")
		if errors.IsConflict(err) {
			return err
		}
		gc.Log.Info().Msg("re-applying updates on latest version and retrying update")
		err = gc.reapplyUpdate()
		if err != nil {
			gc.Log.Error().Err(err).Msg("failed to re-apply update")
			return err
		}
	}
	gc.Log.Info().Msg("gateway updated successfully")
	time.Sleep(1 * time.Second)
	return nil
}

// reapplyUpdate by fetching a new version of the sensor and updating the status
func (gc *GatewayConfig) reapplyUpdate() error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		gwClient := gc.gwcs.ArgoprojV1alpha1().Gateways(gc.Namespace)
		gw, err := gwClient.Get(gc.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		gc.gw.Status = gw.Status
		gc.gw, err = gwClient.Update(gc.gw)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
}

// K8Event returns a kubernetes event
func (gc *GatewayConfig) K8Event(reason string, action gwv1alpha1.NodePhase, configName string) *corev1.Event {
	return &corev1.Event{
		Reason: reason,
		Type: gateway.Kind,
		Action: string(action),
		EventTime: metav1.MicroTime{
			Time: time.Now(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gc.Namespace,
			Name: gc.Name + "-" + common.RandomStringGenerator(),
			Labels: map[string]string{
				common.LabelGatewayConfigurationName: configName,
				common.LabelGatewayEventSeen : "",
				common.LabelGatewayName: gc.Name,
			},
		},
		InvolvedObject: corev1.ObjectReference{
			Namespace: gc.Namespace,
			Name: gc.Name,
			Kind: gateway.Kind,
		},
		Source: corev1.EventSource{
			Component: gc.Name,
		},
		ReportingInstance: "argo-events",
		ReportingController: common.DefaultGatewayControllerDeploymentName,
	}
}

// createK8Event creates a kubernetes event.
func (gc *GatewayConfig) CreateK8Event(event *corev1.Event) error {
	gc.Log.Info().Str("event-action", event.Action).Msg("creating an event to update gateway state")
	_, err := gc.Clientset.CoreV1().Events(gc.Namespace).Create(event)
	return err
}

// GatewayCleanup marks configuration as non-active and marks final gateway state
func (gc *GatewayConfig) GatewayCleanup(config *ConfigContext, errMessage string, err error) {
	var event *corev1.Event
	// mark configuration as deactivated so gateway processor client won't run configStopper in case if there
	// was configuration error.
	config.Active = false
	// check if gateway configuration is in error condition.
	if err != nil {
		gc.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg(errMessage)
		// create k8 event for error state
		event = gc.K8Event(errMessage, gwv1alpha1.NodePhaseError, config.Data.Src)

	} else {
		// gateway successfully completed/deactivated this configuration.
		gc.Log.Info().Str("config-key", config.Data.Src).Msg("configuration completed")
		// create k8 event for completion state
		event = gc.K8Event("configuration completed", gwv1alpha1.NodePhaseCompleted, config.Data.Src)
	}
	err = gc.CreateK8Event(event)
	if err != nil {
		gc.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to create gateway k8 event")
	}
}
