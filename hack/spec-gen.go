package main

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/go-openapi/spec"

	"fmt"
	"k8s.io/kube-openapi/pkg/builder"
	"encoding/json"
	"io/ioutil"
	"k8s.io/kube-openapi/pkg/common"
	"github.com/argoproj/argo-events/gateways/core/webhook"
	"github.com/argoproj/argo-events/gateways/core/resource"
	"github.com/argoproj/argo-events/gateways/core/artifact"
	"github.com/argoproj/argo-events/gateways/core/calendar"
	"github.com/argoproj/argo-events/gateways/core/file"
	"github.com/argoproj/argo-events/gateways/custom/storagegrid"
	"github.com/argoproj/argo-events/gateways/core/stream/nats"
	"github.com/argoproj/argo-events/gateways/core/stream/kafka"
	"github.com/argoproj/argo-events/gateways/core/stream/amqp"
	"github.com/argoproj/argo-events/gateways/core/stream/mqtt"
)

const (
	SpecDir = "swagger-spec"
)

func swaggerSpecs(swaggerFilename string, defNames []string, config *common.Config) {
	fmt.Println(swaggerFilename)
	swagger, err := builder.BuildOpenAPIDefinitionsForResources(config, defNames...)
	if err != nil {
		panic(err)
	}
	// Marshal the swagger spec into JSON, then write it out.
	specBytes, err := json.MarshalIndent(swagger, " ", " ")
	if err != nil {
		panic(fmt.Sprintf("json marshal error: %s", err.Error()))
	}
	err = ioutil.WriteFile(swaggerFilename, specBytes, 0644)
	if err != nil {
		panic(err)
	}
}

func gatewayDefs() {
	var defNames []string
	for name, _ := range v1alpha1.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = v1alpha1.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "gateway.json"), defNames, config)
}

func webhookDefs() {
	var defNames []string
	for name, _ := range webhook.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = webhook.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "webhook.json"), defNames, config)
}

func artifactDefs() {
	var defNames []string
	for name, _ := range artifact.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = artifact.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "artifact.json"), defNames, config)
}

func calendarDefs() {
	var defNames []string
	for name, _ := range calendar.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = calendar.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "calendar.json"), defNames, config)
}

func fileDefs() {
	var defNames []string
	for name, _ := range file.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = file.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "file.json"), defNames, config)
}

func resourceDefs() {
	var defNames []string
	for name, _ := range resource.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = resource.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "resource.json"), defNames, config)
}

func natsDefs() {
	var defNames []string
	for name, _ := range nats.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = nats.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "nats.json"), defNames, config)
}


func kafkaDefs() {
	var defNames []string
	for name, _ := range kafka.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = kafka.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "kafka.json"), defNames, config)
}

func amqpDefs() {
	var defNames []string
	for name, _ := range amqp.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = amqp.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "amqp.json"), defNames, config)
}

func mqttDefs() {
	var defNames []string
	for name, _ := range mqtt.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = mqtt.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "mqtt.json"), defNames, config)
}


func storageGridDefs() {
	var defNames []string
	for name, _ := range storagegrid.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = storagegrid.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", SpecDir, "storagegrid.json"), defNames, config)
}


func main() {
	gatewayDefs()
	webhookDefs()
	artifactDefs()
	calendarDefs()
	fileDefs()
	resourceDefs()
	natsDefs()
	kafkaDefs()
	amqpDefs()
	mqttDefs()
	storageGridDefs()
}

// CreateOpenAPIBuilderConfig hard-codes some values in the API builder
// config for testing.
func createOpenAPIBuilderConfig() *common.Config {
	return &common.Config{
		ProtocolList:   []string{"https"},
		IgnorePrefixes: []string{"/swaggerapi"},
		Info: &spec.Info{
			InfoProps: spec.InfoProps{
				Title:   "Argo-Events",
				Version: "1.0",
			},
		},
	}
}