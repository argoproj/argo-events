package reconciler

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"

	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

type GlobalConfig struct {
	EventBus *EventBusConfig `json:"eventBus"`
}

type EventBusConfig struct {
	NATS      *StanConfig      `json:"nats"`
	JetStream *JetStreamConfig `json:"jetstream"`
}

type StanConfig struct {
	Versions []StanVersion `json:"versions"`
}

type StanVersion struct {
	Version              string `json:"version"`
	NATSStreamingImage   string `json:"natsStreamingImage"`
	MetricsExporterImage string `json:"metricsExporterImage"`
}

type JetStreamConfig struct {
	Settings     string             `json:"settings"`
	StreamConfig string             `json:"streamConfig"`
	Versions     []JetStreamVersion `json:"versions"`
}

type JetStreamVersion struct {
	Version              string `json:"version"`
	NatsImage            string `json:"natsImage"`
	ConfigReloaderImage  string `json:"configReloaderImage"`
	MetricsExporterImage string `json:"metricsExporterImage"`
	StartCommand         string `json:"startCommand"`
}

func (g *GlobalConfig) supportedSTANVersions() []string {
	result := []string{}
	if g.EventBus == nil || g.EventBus.NATS == nil {
		return result
	}
	for _, v := range g.EventBus.NATS.Versions {
		result = append(result, v.Version)
	}
	return result
}

func (g *GlobalConfig) supportedJetStreamVersions() []string {
	result := []string{}
	if g.EventBus == nil || g.EventBus.JetStream == nil {
		return result
	}
	for _, v := range g.EventBus.JetStream.Versions {
		result = append(result, v.Version)
	}
	return result
}

func (g *GlobalConfig) GetSTANVersion(version string) (*StanVersion, error) {
	if g.EventBus == nil || g.EventBus.NATS == nil {
		return nil, fmt.Errorf("\"eventBus.nats\" not found in the configuration")
	}
	if len(g.EventBus.NATS.Versions) == 0 {
		return nil, fmt.Errorf("nats streaming version configuration not found")
	}
	for _, r := range g.EventBus.NATS.Versions {
		if r.Version == version {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("unsupported version %q, supported versions: %q", version, strings.Join(g.supportedSTANVersions(), ","))
}

func (g *GlobalConfig) GetJetStreamVersion(version string) (*JetStreamVersion, error) {
	if g.EventBus == nil || g.EventBus.JetStream == nil {
		return nil, fmt.Errorf("\"eventBus.jetstream\" not found in the configuration")
	}
	if len(g.EventBus.JetStream.Versions) == 0 {
		return nil, fmt.Errorf("jetstream version configuration not found")
	}
	for _, r := range g.EventBus.JetStream.Versions {
		if r.Version == version {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("unsupported version %q, supported versions: %q", version, strings.Join(g.supportedJetStreamVersions(), ","))
}

func LoadConfig(onErrorReloading func(error)) (*GlobalConfig, error) {
	v := sharedutil.ViperWithLogging()
	v.SetConfigName("controller-config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/argo-events")
	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration file. %w", err)
	}
	r := &GlobalConfig{}
	err = v.Unmarshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal configuration file. %w", err)
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		err = v.Unmarshal(r)
		if err != nil {
			onErrorReloading(err)
		}
	})
	return r, nil
}

func ValidateConfig(config *GlobalConfig) error {
	if len(config.supportedJetStreamVersions()) == 0 {
		return fmt.Errorf("no jetstream versions were provided in the controller config")
	}

	if len(config.supportedSTANVersions()) == 0 {
		return fmt.Errorf("no stan versions were provided in the controller config")
	}

	return nil
}
