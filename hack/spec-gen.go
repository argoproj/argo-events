package main

import (
	salphav1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	galphav1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"

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
	"os"
)

var specDir = getSpecDir() + "/hack/swagger-spec"

func getSpecDir() string {
	specDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return specDir
}

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

func sensorDefs() {
	var defNames []string
	for name, _ := range salphav1.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = salphav1.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "sensor.json"), defNames, config)
}


func gatewayDefs() {
	var defNames []string
	for name, _ := range galphav1.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}
	config := createOpenAPIBuilderConfig()
	config.GetDefinitions = galphav1.GetOpenAPIDefinitions
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "gateway.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "webhook.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "artifact.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "calendar.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "file.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "resource.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "nats.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "kafka.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "amqp.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "mqtt.json"), defNames, config)
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
	swaggerSpecs(fmt.Sprintf("%s/%s", specDir, "storagegrid.json"), defNames, config)
}


func main() {
	sensorDefs()
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
				Version: "v0.6",
			},
		},
	}
}