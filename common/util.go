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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/blackrock/axis/pkg/apis/sensor"
)

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
	if errors.IsNotFound(err) || errors.IsForbidden(err) || errors.IsInvalid(err) || errors.IsMethodNotSupported(err) {
		return false
	}
	return true
}

// AddJobAnnotation adds an annotation to a job
func AddJobAnnotation(c kubernetes.Interface, jobName, namespace, key, value string) error {
	return addJobMetadata(c, "annotations", jobName, namespace, key, value)
}

// AddJobLabel adds an label to a job
func AddJobLabel(c kubernetes.Interface, jobName, namespace, key, value string) error {
	return addJobMetadata(c, "labels", jobName, namespace, key, value)
}

// addJobMetadata is helper to either add a job label or annotation to the job
func addJobMetadata(c kubernetes.Interface, field, jobName, namespace, key, value string) error {
	metadata := map[string]interface{}{
		"metadata": map[string]interface{}{
			field: map[string]string{
				key: value,
			},
		},
	}
	var err error
	patch, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 5; attempt++ {
		_, err = c.BatchV1().Jobs(namespace).Patch(jobName, types.MergePatchType, patch)
		if err != nil {
			if !errors.IsConflict(err) {
				return err
			}
		} else {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return err
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

// CreateJobPrefix from the name of the sensor by simply attaching the string '-sensor'
// job name follows format: {sensorName}-sensor
func CreateJobPrefix(name string) string {
	return name + "-" + sensor.Singular
}

// ParseJobPrefix and return the sensorName
func ParseJobPrefix(job string) string {
	// first trim the ID at the end, should be everything after and including the last '-'
	withoutUniqueID := job[:strings.LastIndex(job, "-")]
	return strings.TrimSuffix(withoutUniqueID, "-"+sensor.Singular)
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
