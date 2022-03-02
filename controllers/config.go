package controllers

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type GlobalConfig struct {
	EventBus *EventBusConfig `json:"eventBus"`
}

type EventBusConfig struct {
	NATS      *NatsStreamingConfig `json:"nats"`
	JetStream *JetStreamConfig     `json:"jetstream"`
}

type NatsStreamingConfig struct {
	Versions []NatsStreamingVersion `json:"versions"`
}

type NatsStreamingVersion struct {
	Version              string `json:"version"`
	NatsStreamingImage   string `json:"natsStreamingImage"`
	MetricsExporterImage string `json:"metricsExporterImage"`
}

type JetStreamConfig struct {
	Settings string             `json:"settings"`
	Versions []JetStreamVersion `json:"versions"`
}

type JetStreamVersion struct {
	Version              string `json:"version"`
	NatsImage            string `json:"natsImage"`
	MetricsExporterImage string `json:"metricsExporterImage"`
	StartCommand         string `json:"startCommand"`
}

func (g *GlobalConfig) GetNatsStreamingVersion(version string) (*NatsStreamingVersion, error) {
	if g.EventBus.NATS == nil || len(g.EventBus.NATS.Versions) == 0 {
		return nil, fmt.Errorf("nats streaming configuration not found")
	}
	for _, r := range g.EventBus.NATS.Versions {
		if r.Version == version {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("no nats streaming configuration found for %q", version)
}

func (g *GlobalConfig) GetJetStreamVersion(version string) (*JetStreamVersion, error) {
	if g.EventBus.JetStream == nil || len(g.EventBus.JetStream.Versions) == 0 {
		return nil, fmt.Errorf("jetstream configuration not found")
	}
	for _, r := range g.EventBus.JetStream.Versions {
		if r.Version == version {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("no jetstream configuration found for %q", version)
}

func LoadConfig(onErrorReloading func(error)) (*GlobalConfig, error) {
	v := viper.New()
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
