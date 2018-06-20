package main

import (
	"fmt"
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/shared"
	"github.com/argoproj/argo-events/signals/nats"
	plugin "github.com/hashicorp/go-plugin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	namespace     string
	config        *rest.Config
	kubeClientset kubernetes.Interface
)

func init() {
	var ok bool
	namespace, ok = os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		panic(fmt.Errorf("Unable to get namespace from environment variable %s", common.EnvVarNamespace))
	}

	var err error
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	config, err = common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	kubeClientset = kubernetes.NewForConfigOrDie(config)
}

func main() {
	// these are the signals
	nats := nats.New()
	/*
		mqtt := mqtt.New()
		kafka := kafka.New()
		amqp := amqp.New()
		artifact := artifact.New(nats, kubeClientset, namespace)
		cal := calendar.New()
		resource := resource.New(config)
		web := webhook.New()
	*/

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			"NATS": shared.NewPlugin(nats),
			//"MQTT":     shared.NewPlugin(mqtt),
			//"KAFKA":    shared.NewPlugin(kafka),
			//"AMQP":     shared.NewPlugin(amqp),
			//"Artifact": shared.NewArtifactPlugin(artifact),
			//"Calendar": shared.NewPlugin(cal),
			//"Resource": shared.NewPlugin(resource),
			//"Webhook":  shared.NewPlugin(web),
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
