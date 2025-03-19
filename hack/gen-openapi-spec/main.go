package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"

	aeopenapi "github.com/argoproj/argo-events/pkg/apis/events/openapi"
)

type (
	obj = map[string]interface{}
)

// Generate OpenAPI spec definitions for Workflow Resource
func main() {
	if len(os.Args) <= 3 {
		log.Fatal("Supply a version")
	}
	log.Println(os.Args)
	version := os.Args[1]
	kubeSwaggerPath := os.Args[2]
	output := os.Args[3]
	if version != "latest" && !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	referenceCallback := func(name string) spec.Ref {
		return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name)))
	}
	defs := spec.Definitions{}
	dependencies := []string{}
	for defName, val := range aeopenapi.GetOpenAPIDefinitions(referenceCallback) {
		defs[swaggify(defName)] = val.Schema
		dependencies = append(dependencies, val.Dependencies...)
	}

	k8sDefinitions := getKubernetesSwagger(kubeSwaggerPath)
	for _, dep := range dependencies {
		if !strings.Contains(dep, "k8s.io") {
			continue
		}
		d := swaggify(dep)
		if kd, ok := k8sDefinitions[d]; ok {
			defs[d] = kd
		}
	}
	for d, s := range k8sDefinitions {
		defs[d] = s
	}

	swagger := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:     "2.0",
			Definitions: defs,
			Paths:       &spec.Paths{Paths: map[string]spec.PathItem{}},
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:   "Argo Events",
					Version: version,
				},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		log.Fatal(err.Error())
	}
	err = os.WriteFile(output, jsonBytes, 0644)
	if err != nil {
		panic(err)
	}
	f, err := os.Open(output)
	if err != nil {
		panic(err)
	}
	// filter out "default" fields from swagger definitions properties because they are being set to empty strings and it makes the swagger validation fail.
	swaggerObj := obj{}
	err = json.NewDecoder(f).Decode(&swaggerObj)
	if err != nil {
		panic(err)
	}
	definitions := swaggerObj["definitions"].(obj)

	for _, d := range definitions {
		props, ok := d.(obj)["properties"].(obj)
		if ok {
			for _, prop := range props {
				prop := prop.(obj)
				delete(prop, "default")
				items, ok := prop["items"].(obj)
				if ok {
					delete(items, "default")
				}
				additionalProperties, ok := prop["additionalProperties"].(obj)
				if ok {
					delete(additionalProperties, "default")
				}
			}
		}
		props, ok = d.(obj)["additionalProperties"].(obj)
		if ok {
			delete(props, "default")
		}
	}

	f, err = os.Create(output)
	if err != nil {
		panic(err)
	}
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	err = e.Encode(swaggerObj)
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}
}

// swaggify converts the github package
// e.g.:
// github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1.Sensor
// to:
// io.argoproj.v1alpha1.Sensor
func swaggify(name string) string {
	name = strings.ReplaceAll(name, "github.com/argoproj/argo-events/pkg/apis", "argoproj.io")
	parts := strings.Split(name, "/")
	hostParts := strings.Split(parts[0], ".")
	// reverses something like k8s.io to io.k8s
	for i, j := 0, len(hostParts)-1; i < j; i, j = i+1, j-1 {
		hostParts[i], hostParts[j] = hostParts[j], hostParts[i]
	}
	parts[0] = strings.Join(hostParts, ".")
	return strings.Join(parts, ".")
}

func getKubernetesSwagger(kubeSwaggerPath string) spec.Definitions {
	data, err := os.ReadFile(kubeSwaggerPath)
	if err != nil {
		panic(err)
	}
	swagger := &spec.Swagger{}
	err = json.Unmarshal(data, swagger)
	if err != nil {
		panic(err)
	}
	return swagger.Definitions
}
