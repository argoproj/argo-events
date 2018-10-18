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

package common

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
)

// DefaultConfigMapName returns a formulated name for a configmap name based on the sensor-controller deployment name
func DefaultConfigMapName(controllerName string) string {
	return fmt.Sprintf("%s-configmap", controllerName)
}

// DefaultSensorDeploymentName returns a formulated name for sensor deployment
func DefaultSensorDeploymentName(deploymentName string) string {
	return fmt.Sprintf("%s-sensor-deployement", deploymentName)
}

// DefaultSensorJobName returns a formulated name for a sensor job
func DefaultSensorJobName(jobName string) string {
	return fmt.Sprintf("%s-sensor-job", jobName)
}

// DefaultGatewayDeploymentName returns a formulated name for a gateway deployment
func DefaultGatewayDeploymentName(deploymentName string) string {
	return fmt.Sprintf("%s-gateway-deployment", deploymentName)
}

// DefaultGatewayTransformerConfigMapName returns a formulated name for a gateway transformer configmap
func DefaultGatewayTransformerConfigMapName(configMapName string) string {
	return fmt.Sprintf("%s-gateway-transformer-configmap", configMapName)
}

// DefaultGatewayServiceName returns a formulated name for a gateway service
func DefaultGatewayServiceName(serviceName string) string {
	return fmt.Sprintf("%s-gateway-svc", serviceName)
}

// DefaultSensorServiceName returns a formulated name for a sensor service
func DefaultSensorServiceName(serviceName string) string {
	return fmt.Sprintf("%s-sensor-svc", serviceName)
}

func DefaultGatewayConfigurationName(gatewayName string, configurationName string) string {
	return fmt.Sprintf("%s/%s", gatewayName, configurationName)
}

// GetClientConfig return rest config, if path not specified, assume in cluster config
func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// ServerResourceForGroupVersionKind finds the API resources that fit the GroupVersionKind schema
func ServerResourceForGroupVersionKind(disco discovery.DiscoveryInterface, gvk schema.GroupVersionKind) (*metav1.APIResource, error) {
	resources, err := disco.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return nil, err
	}
	for _, r := range resources.APIResources {
		if r.Kind == gvk.Kind {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("server is unable to handle %s", gvk)
}

// SendSuccessResponse sends http success response
func SendSuccessResponse(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(SuccessResponse))
}

// SendErrorResponse sends http error response
func SendErrorResponse(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusBadRequest)
	writer.Write([]byte(ErrorResponse))
}
