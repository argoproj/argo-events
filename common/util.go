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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClientConfig return rest config, if path not specified, assume in cluster config
func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// SendSuccessResponse sends http success response
func SendSuccessResponse(writer http.ResponseWriter, response string) {
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write([]byte(response)); err != nil {
		fmt.Printf("failed to write the response. err: %+v\n", err)
	}
}

// SendErrorResponse sends http error response
func SendErrorResponse(writer http.ResponseWriter, response string) {
	writer.WriteHeader(http.StatusBadRequest)
	if _, err := writer.Write([]byte(response)); err != nil {
		fmt.Printf("failed to write the response. err: %+v\n", err)
	}
}

// SendInternalErrorResponse sends http internal error response
func SendInternalErrorResponse(writer http.ResponseWriter, response string) {
	writer.WriteHeader(http.StatusInternalServerError)
	if _, err := writer.Write([]byte(response)); err != nil {
		fmt.Printf("failed to write the response. err: %+v\n", err)
	}
}

// SendResponse sends http response with given status code
func SendResponse(writer http.ResponseWriter, statusCode int, response string) {
	writer.WriteHeader(statusCode)
	if _, err := writer.Write([]byte(response)); err != nil {
		fmt.Printf("failed to write the response. err: %+v\n", err)
	}
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

// GetSecretValue retrieves the secret value from the secret in namespace with name and key
func GetSecretValue(client kubernetes.Interface, namespace string, selector *v1.SecretKeySelector) (string, error) {
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

// GetEnvFromSecret retrieves the value of envFrom.secretRef
// "${secretRef.name}_" is expected to be defined as "prefix"
func GetEnvFromSecret(selector *v1.SecretKeySelector) (string, bool) {
	return os.LookupEnv(fmt.Sprintf("%s_%s", selector.Name, selector.Key))
}

// GenerateEnvFromSecretSpec builds a "envFrom" spec with a secretKeySelector
func GenerateEnvFromSecretSpec(selector *v1.SecretKeySelector) v1.EnvFromSource {
	return v1.EnvFromSource{
		Prefix: selector.Name + "_",
		SecretRef: &v1.SecretEnvSource{
			LocalObjectReference: v1.LocalObjectReference{
				Name: selector.Name,
			},
		},
	}
}

// GetSecretFromVolume retrieves the value of mounted secret volume
// "/argo-events/secrets/${secretRef.name}/${secretRef.key}" is expected to be the file path
func GetSecretFromVolume(selector *v1.SecretKeySelector) (string, error) {
	filePath := fmt.Sprintf("/argo-events/secrets/%s/%s", selector.Name, selector.Key)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get secret value of name: %s, key: %s", selector.Name, selector.Key)
	}
	return string(data), nil
}

// GetConfigMapFromVolume retrieves the value of mounted config map volume
// "/argo-events/config/${configMapRef.name}/${configMapRef.key}" is expected to be the file path
func GetConfigMapFromVolume(selector *v1.ConfigMapKeySelector) (string, error) {
	filePath := fmt.Sprintf("/argo-events/config/%s/%s", selector.Name, selector.Key)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get configMap value of name: %s, key: %s", selector.Name, selector.Key)
	}
	return string(data), nil
}

// GetEnvFromConfigMap retrieves the value of envFrom.configMapRef
// "${configMapRef.name}_" is expected to be defined as "prefix"
func GetEnvFromConfigMap(selector *v1.ConfigMapKeySelector) (string, bool) {
	return os.LookupEnv(fmt.Sprintf("%s_%s", selector.Name, selector.Key))
}

// GenerateEnvFromConfigMapSpec builds a "envFrom" spec with a configMapKeySelector
func GenerateEnvFromConfigMapSpec(selector *v1.ConfigMapKeySelector) v1.EnvFromSource {
	return v1.EnvFromSource{
		Prefix: selector.Name + "_",
		ConfigMapRef: &v1.ConfigMapEnvSource{
			LocalObjectReference: v1.LocalObjectReference{
				Name: selector.Name,
			},
		},
	}
}

// GetTLSConfig returns a tls configuration for given cert and key.
func GetTLSConfig(caCertPath, clientCertPath, clientKeyPath string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read ca cert file %s", caCertPath)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)

	clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load client cert key pair %s", caCertPath)
	}
	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{clientCert},
	}, nil
}

// VolumesFromSecretsOrConfigMaps builds volumes and volumeMounts spec based on
// the obj and its children's secretKeyselector or configMapKeySelector
func VolumesFromSecretsOrConfigMaps(obj interface{}, t reflect.Type) ([]v1.Volume, []v1.VolumeMount) {
	resultVolumes := []v1.Volume{}
	resultMounts := []v1.VolumeMount{}
	values := findTypeValues(obj, t)
	if len(values) == 0 {
		return resultVolumes, resultMounts
	}
	switch t {
	case SecretKeySelectorType:
		for _, v := range values {
			selector := v.(*v1.SecretKeySelector)
			vol, mount := GenerateSecretVolumeSpecs(selector)
			resultVolumes = append(resultVolumes, vol)
			resultMounts = append(resultMounts, mount)
		}
	case ConfigMapKeySelectorType:
		for _, v := range values {
			selector := v.(*v1.ConfigMapKeySelector)
			vol, mount := GenerateConfigMapVolumeSpecs(selector)
			resultVolumes = append(resultVolumes, vol)
			resultMounts = append(resultMounts, mount)
		}
	default:
	}
	return uniqueVolumes(resultVolumes), uniqueVolumeMounts(resultMounts)
}

// Find all the values obj's children matching provided type, type needs to be a pointer
func findTypeValues(obj interface{}, t reflect.Type) []interface{} {
	result := []interface{}{}
	value := reflect.ValueOf(obj)
	findTypesRecursive(&result, value, t)
	return result
}

func findTypesRecursive(result *[]interface{}, obj reflect.Value, t reflect.Type) {
	if obj.Type() == t && obj.CanInterface() && !obj.IsNil() {
		*result = append(*result, obj.Interface())
	}
	switch obj.Kind() {
	case reflect.Ptr:
		objValue := obj.Elem()
		// Check if it is nil
		if !objValue.IsValid() {
			return
		}
		findTypesRecursive(result, objValue, t)
	case reflect.Interface:
		objValue := obj.Elem()
		// Check if it is nil
		if !objValue.IsValid() {
			return
		}
		findTypesRecursive(result, objValue, t)
	case reflect.Struct:
		for i := 0; i < obj.NumField(); i++ {
			if obj.Field(i).CanInterface() {
				findTypesRecursive(result, obj.Field(i), t)
			}
		}
	case reflect.Slice:
		for i := 0; i < obj.Len(); i++ {
			findTypesRecursive(result, obj.Index(i), t)
		}
	case reflect.Map:
		iter := obj.MapRange()
		for iter.Next() {
			findTypesRecursive(result, iter.Value(), t)
		}
	default:
		return
	}
}

// GenerateSecretVolumeSpecs builds a "volume" and "volumeMount"spec with a secretKeySelector
func GenerateSecretVolumeSpecs(selector *v1.SecretKeySelector) (v1.Volume, v1.VolumeMount) {
	volName := strings.ReplaceAll("secret-"+selector.Name, "_", "-")
	return v1.Volume{
			Name: volName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: selector.Name,
				},
			},
		}, v1.VolumeMount{
			Name:      volName,
			ReadOnly:  true,
			MountPath: "/argo-events/secrets/" + selector.Name,
		}
}

// GenerateConfigMapVolumeSpecs builds a "volume" and "volumeMount"spec with a configMapKeySelector
func GenerateConfigMapVolumeSpecs(selector *v1.ConfigMapKeySelector) (v1.Volume, v1.VolumeMount) {
	volName := strings.ReplaceAll("cm-"+selector.Name, "_", "-")
	return v1.Volume{
			Name: volName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: selector.Name,
					},
				},
			},
		}, v1.VolumeMount{
			Name:      volName,
			ReadOnly:  true,
			MountPath: "/argo-events/config/" + selector.Name,
		}
}

func uniqueVolumes(vols []v1.Volume) []v1.Volume {
	rVols := []v1.Volume{}
	keys := make(map[string]bool)
	for _, e := range vols {
		if _, value := keys[e.Name]; !value {
			keys[e.Name] = true
			rVols = append(rVols, e)
		}
	}
	return rVols
}

func uniqueVolumeMounts(mounts []v1.VolumeMount) []v1.VolumeMount {
	rMounts := []v1.VolumeMount{}
	keys := make(map[string]bool)
	for _, e := range mounts {
		if _, value := keys[e.Name]; !value {
			keys[e.Name] = true
			rMounts = append(rMounts, e)
		}
	}
	return rMounts
}
