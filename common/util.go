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
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DefaultConfigMapName returns a formulated name for a configmap name based on the sensor-controller deployment name
func DefaultConfigMapName(controllerName string) string {
	return fmt.Sprintf("%s-configmap", controllerName)
}

// ServiceDNSName returns a formulated dns name for a service
func ServiceDNSName(serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)
}

// DefaultEventSourceName returns a formulated name for a gateway configuration
func DefaultEventSourceName(gatewayName string, configurationName string) string {
	return fmt.Sprintf("%s:%s", gatewayName, configurationName)
}

// DefaultNatsQueueName returns a queue name for nats subject
func DefaultNatsQueueName(subject string) string {
	return fmt.Sprintf("%s-%s", subject, "queue")
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
func SendSuccessResponse(writer http.ResponseWriter, response string) {
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(response))
}

// SendErrorResponse sends http error response
func SendErrorResponse(writer http.ResponseWriter, response string) {
	writer.WriteHeader(http.StatusBadRequest)
	writer.Write([]byte(response))
}

// SendInternalErrorResponse sends http internal error response
func SendInternalErrorResponse(writer http.ResponseWriter, response string) {
	writer.WriteHeader(http.StatusInternalServerError)
	writer.Write([]byte(response))
}

// Hasher hashes a string
func Hasher(value string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return fmt.Sprintf("%v", h.Sum32())
}

// GetObjectHash returns hash of a given object
func GetObjectHash(obj metav1.Object) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resource")
	}
	return Hasher(string(b)), nil
}

// FormatEndpoint returns a formatted api endpoint
func FormatEndpoint(endpoint string) string {
	if !strings.HasPrefix(endpoint, "/") {
		return fmt.Sprintf("/%s", endpoint)
	}
	return endpoint
}

// FormattedURL returns a formatted url
func FormattedURL(url, endpoint string) string {
	return fmt.Sprintf("%s%s", url, FormatEndpoint(endpoint))
}

func ErrEventSourceTypeMismatch(eventSourceType string) string {
	return fmt.Sprintf("event source is not type of %s", eventSourceType)
}

// GetSecrets retrieves the secret value from the secret in namespace with name and key
func GetSecrets(client kubernetes.Interface, namespace string, selector *v1.SecretKeySelector) (string, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(selector.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := secret.Data[selector.Key]
	if !ok {
		return "", errors.Errorf("secret '%s' does not have the key '%s'", selector.Name, selector.Key)
	}
	return string(val), nil
}
