package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	cv1 "github.com/argoproj/argo-events/pkg/apis/common"
	gwv1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/go-openapi/spec"
	"k8s.io/kube-openapi/pkg/common"
)

// Generate OpenAPI spec definitions for Workflow Resource
func main() {
	if len(os.Args) <= 1 {
		log.Fatal("Supply a version")
	}
	version := os.Args[1]
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	gwOAPIDefs := gwv1.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name)))
	})
	sOAPIDefs := sv1.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name)))
	})
	cOAPIDefs := cv1.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name)))
	})
	defs := spec.Definitions{}
	for defName, val := range gwOAPIDefs {
		defs[swaggify(defName)] = val.Schema
	}
	for defName, val := range sOAPIDefs {
		defs[swaggify(defName)] = val.Schema
	}
	for defName, val := range cOAPIDefs {
		defs[swaggify(defName)] = val.Schema
	}
	swagger := spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:     "2.0",
			Definitions: defs,
			Paths:       &spec.Paths{Paths: map[string]spec.PathItem{}},
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:   "Argo",
					Version: version,
				},
			},
		},
	}
	jsonBytes, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(string(jsonBytes))
}

// swaggify converts the github package
// e.g.:
// github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1.Gateway
// to:
// io.argoproj.workflow.v1alpha1.Workflow
func swaggify(name string) string {
	name = strings.Replace(name, "github.com/argoproj/argo-events/pkg/apis", "argoproj.io", -1)
	parts := strings.Split(name, "/")
	hostParts := strings.Split(parts[0], ".")
	// reverses something like k8s.io to io.k8s
	for i, j := 0, len(hostParts)-1; i < j; i, j = i+1, j-1 {
		hostParts[i], hostParts[j] = hostParts[j], hostParts[i]
	}
	parts[0] = strings.Join(hostParts, ".")
	return strings.Join(parts, ".")
}
