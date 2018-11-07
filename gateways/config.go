package gateways

import (
	"context"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwclientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	zlog "github.com/rs/zerolog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"github.com/argoproj/argo-events/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"time"
)

// GatewayConfig provides a generic configuration for a gateway
type GatewayConfig struct {
	// Log provides fast and simple logger dedicated to JSON output
	Log zlog.Logger
	// Clientset is client for kubernetes API
	Clientset kubernetes.Interface
	// Name is gateway name
	Name string
	// Namespace is namespace for the gateway to run inside
	Namespace string
	// KubeConfig rest client config
	KubeConfig *rest.Config
	// gateway holds Gateway custom resource
	gw *v1alpha1.Gateway
	// gwClientset is gateway clientset
	gwcs gwclientset.Interface
	// transformerPort is gateway transformer port to dispatch event to
	transformerPort string
	// registeredConfigs stores information about current configurations that are running in the gateway
	registeredConfigs map[string]*ConfigContext
	// configName is name of configmap that contains run configuration/s for the gateway
	configName string
	// controllerInstanceId is instance ID of the gateway controller
	controllerInstanceID string
}

// ConfigContext contains information of a configuration for gateway to run.
type ConfigContext struct {
	// Data holds the actual configuration
	Data *ConfigData

	StartChan chan struct{}
	// StopCh is used to send a stop signal to configuration runner/executor
	StopChan chan struct{}

	DataChan chan []byte

	DoneChan chan struct{}

	ErrChan chan error

	// Active tracks configuration state as running or stopped
	Active bool
	// Cancel is called to cancel the context used by client to communicate with gRPC server.
	// Use it only if gateway is implemented as gRPC server.
	Cancel context.CancelFunc
}

// ConfigData holds the actual configuration
type ConfigData struct {
	// Unique ID for configuration
	ID string `json:"id"`
	// TimeID is unique time id for configuration. It is used to resolve conflict between two k8 events arriving in different oder.
	// Consider a configuration becomes stale, then as the configuration is stopped gateway processor server will generate Completed phase event,
	// meanwhile gateway processor will generate Remove phase event, if remove is generated before complete and if complete ohase event is consumed first
	// then it will disregard the remove event. this is not what is expected. TimeID will be used as override in such scenarios.
	// This is because k8 events does not have temporal order.
	TimeID string `json:"timeID"`
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
	// HTTPServerPort is the port on which gateway processor server is running
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

// ConfigExecutor is interface a gateway processor server must implement
type ConfigExecutor interface {
	StartConfig(configContext *ConfigContext)
	StopConfig(configContext *ConfigContext)
	Validate(configContext *ConfigContext) error
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
	log := zlog.New(os.Stdout).With().Str("gateway-name", name).Caller().Logger()
	namespace, ok := os.LookupEnv(common.GatewayNamespace)
	if !ok {
		log.Panic().Str("gateway-name", name).Err(err).Msg("no namespace provided")
	}
	transformerPort, ok := os.LookupEnv(common.EnvVarGatewayTransformerPort)
	if !ok {
		log.Panic().Str("gateway-name", name).Err(err).Msg("gateway transformer port is not provided")
	}
	configName, ok := os.LookupEnv(common.EnvVarGatewayProcessorConfigMap)
	if !ok {
		log.Panic().Str("gateway-name", name).Err(err).Msg("gateway processor configmap is not provided")
	}
	controllerInstanceID, ok := os.LookupEnv(common.EnvVarGatewayControllerInstanceID)
	if !ok {
		log.Panic().Str("gateway-name", name).Err(err).Msg("gateway controller instance ID is not provided")
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gwcs := gwclientset.NewForConfigOrDie(restConfig)
	gw, err := gwcs.ArgoprojV1alpha1().Gateways(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Panic().Str("gateway-name", name).Err(err).Msg("failed to get gateway resource")
	}
	return &GatewayConfig{
		Log:                  log,
		Clientset:            clientset,
		Namespace:            namespace,
		Name:                 name,
		KubeConfig:           restConfig,
		registeredConfigs:    make(map[string]*ConfigContext),
		transformerPort:      transformerPort,
		configName:           configName,
		gwcs:                 gwcs,
		gw:                   gw,
		controllerInstanceID: controllerInstanceID,
	}
}

// NewHTTPGatewayServerConfig returns a new HTTPGatewayServerConfig
func NewHTTPGatewayServerConfig() *HTTPGatewayServerConfig {
	httpGatewayServerConfig := &HTTPGatewayServerConfig{}
	httpGatewayServerConfig.HTTPServerPort = func() string {
		httpServerPort, ok := os.LookupEnv(common.EnvVarGatewayProcessorServerHTTPPort)
		if !ok {
			panic("gateway server http port is not provided")
		}
		return httpServerPort
	}()
	httpGatewayServerConfig.HTTPClientPort = func() string {
		httpClientPort, ok := os.LookupEnv(common.EnvVarGatewayProcessorClientHTTPPort)
		if !ok {
			panic("gateway client http port is not provided")
		}
		return httpClientPort
	}()
	httpGatewayServerConfig.ConfigActivateEndpoint = func() string {
		configActivateEndpoint, ok := os.LookupEnv(common.EnvVarGatewayProcessorHTTPServerConfigStartEndpoint)
		if !ok {
			panic("gateway config activation endpoint is not provided")
		}
		return configActivateEndpoint
	}()
	httpGatewayServerConfig.ConfigurationDeactivateEndpoint = func() string {
		configDeactivateEndpoint, ok := os.LookupEnv(common.EnvVarGatewayProcessorHTTPServerConfigStopEndpoint)
		if !ok {
			panic("gateway config deactivation endpoint is not provided")
		}
		return configDeactivateEndpoint
	}()
	httpGatewayServerConfig.EventEndpoint = func() string {
		eventEndpoint, ok := os.LookupEnv(common.EnvVarGatewayProcessorHTTPServerEventEndpoint)
		if !ok {
			panic("gateway event endpoint is not provided")
		}
		return eventEndpoint
	}()
	httpGatewayServerConfig.GwConfig = NewGatewayConfiguration()
	return httpGatewayServerConfig
}

// createInternalConfigs creates an internal representation of configuration declared in the gateway configmap.
// returned configurations are map of hash of configuration and configuration itself.
// Creating a hash of configuration makes it easy to check equality of two configurations.
func (gc *GatewayConfig) createInternalConfigs(cm *corev1.ConfigMap) (map[string]*ConfigContext, error) {
	configs := make(map[string]*ConfigContext)
	for configKey, configValue := range cm.Data {
		hashKey := Hasher(configKey + configValue)
		gc.Log.Info().Str("config-key", configKey).Str("config-data", configValue).Str("hash", string(hashKey)).Msg("configuration hash")
		currentTimeStr := time.Now().String()
		timeID := Hasher(currentTimeStr)
		configs[hashKey] = &ConfigContext{
			Data: &ConfigData{
				ID:     hashKey,
				TimeID: timeID,
				Src:    configKey,
				Config: configValue,
			},
			StopChan:  make(chan struct{}),
			DataChan:  make(chan []byte),
			StartChan: make(chan struct{}),
			ErrChan:   make(chan error),
			DoneChan:  make(chan struct{}),
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

	for currentConfigKey := range gc.registeredConfigs {
		currentConfigKeys = append(currentConfigKeys, currentConfigKey)
	}
	for updatedConfigKey := range newConfigs {
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

// manageConfigurations syncs registered configurations and updated gateway configmap
func (gc *GatewayConfig) manageConfigurations(executor ConfigExecutor, cm *corev1.ConfigMap) error {
	newConfigs, err := gc.createInternalConfigs(cm)
	if err != nil {
		return err
	}
	staleConfigKeys, newConfigKeys := gc.diffConfigurations(newConfigs)
	gc.Log.Debug().Interface("stale-config-keys", staleConfigKeys).Msg("stale configurations")
	gc.Log.Debug().Interface("new-config-keys", newConfigKeys).Msg("new configurations")

	// run new configurations
	for _, newConfigKey := range newConfigKeys {
		newConfig := newConfigs[newConfigKey]
		gc.registeredConfigs[newConfigKey] = newConfig
		// check if this is reactivation of configuration
		node := gc.getNodeByID(newConfig.Data.ID)
		if node != nil {
			// no need to initialize the node as this is a reactivation
			gc.Log.Info().Str("config-key", newConfig.Data.Src).Msg("reactivating configuration...")
		} else {
			gc.Log.Info().Str("config-key", newConfig.Data.Src).Msg("activating configuration...")
			// create k8 event for new configuration
			newConfigEvent := gc.GetK8Event("new configuration", v1alpha1.NodePhaseInitialized, newConfig.Data)
			_, err = common.CreateK8Event(newConfigEvent, gc.Clientset)
			if err != nil {
				gc.Log.Error().Str("config-name", newConfig.Data.Src).Err(err).Msg("failed to create k8 event to update gateway configurations. skipping configuration...")
				continue
			}
			gc.Log.Info().Str("config-key", newConfig.Data.Src).Msg("created k8 event for new configuration.")
		}

		// validate configuration
		// TODO: If validation of a configuration fails, gateway state should be marked as  Error
		err := executor.Validate(newConfig)
		if err != nil {
			return err
		}

		go func() {
			err := <-newConfig.ErrChan
			gc.GatewayCleanup(newConfig, err)
		}()

		// run configuration
		go executor.StartConfig(newConfig)
	}

	// remove stale configurations
	for _, staleConfigKey := range staleConfigKeys {
		staleConfig := gc.registeredConfigs[staleConfigKey]
		executor.StopConfig(staleConfig)
		if err == nil {
			gc.Log.Info().Str("config", staleConfig.Data.Src).Msg("configuration deactivated.")
			delete(gc.registeredConfigs, staleConfigKey)
			// create a k8 event to remove the node configuration from gateway resource
			removeConfigEvent := gc.GetK8Event("stale configuration", v1alpha1.NodePhaseRemove, staleConfig.Data)
			_, err = common.CreateK8Event(removeConfigEvent, gc.Clientset)
			if err != nil {
				gc.Log.Error().Err(err).Str("config", staleConfig.Data.Src).Msg("failed to create k8 event to remove configuration")
			}
		} else {
			gc.Log.Error().Str("config", staleConfig.Data.Src).Err(err).Msg("failed to deactivate the configuration.")
		}
	}
	return nil
}
