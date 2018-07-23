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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const namespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

var (
	// DefaultSensorControllerNamespace is the default namespace where the sensor controller is installed
	DefaultSensorControllerNamespace = "default"

	// ErrReadNamespace occurs when the namespace cannot be read from a Kubernetes pod's service account token
	ErrReadNamespace = errors.New("Could not read namespace from service account secret")
)

func init() {
	RefreshNamespace()
}

// DefaultRetry is a default retry backoff settings when retrying API calls
var DefaultRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

// IsRetryableKubeAPIError returns if the error is a retryable kubernetes error
func IsRetryableKubeAPIError(err error) bool {
	// get original error if it was wrapped
	if apierr.IsNotFound(err) || apierr.IsForbidden(err) || apierr.IsInvalid(err) || apierr.IsMethodNotSupported(err) {
		return false
	}
	return true
}

// DefaultConfigMapName returns a formulated name for a configmap name based on the sensor-controller deployment name
func DefaultConfigMapName(controllerName string) string {
	return fmt.Sprintf("%s-configmap", controllerName)
}

// GetClientConfig return rest config, if path not specified, assume in cluster config
func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// CreateServiceSuffix formats the service name backed by sensor job
func CreateServiceSuffix(name string) string {
	return name + "-svc"
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
	return nil, fmt.Errorf("Server is unable to handle %s", gvk)
}

// detectNamespace attemps to read the namespace from the mounted service account token
// Note that this will return an error if running outside a Kubernetes pod
func detectNamespace() (string, error) {
	// Make sure it's a file and we can read it
	if s, e := os.Stat(namespacePath); e != nil {
		return "", e
	} else if s.IsDir() {
		return "", ErrReadNamespace
	}

	// Read the file, and cast to a string
	ns, e := ioutil.ReadFile(namespacePath)
	return string(ns), e
}

// RefreshNamespace performs waterfall logic for choosing a "default" namespace
// this function is run as part of an init() function
func RefreshNamespace() {
	// 1 - env variable
	nm, ok := os.LookupEnv(EnvVarNamespace)
	if ok {
		DefaultSensorControllerNamespace = nm
		return
	}

	// 2 - pod service account token
	nm, err := detectNamespace()
	if err == nil {
		DefaultSensorControllerNamespace = nm
	}

	// 3 - use the DefaultSensorControllerNamespace
	return
}
