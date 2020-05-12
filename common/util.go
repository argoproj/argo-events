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
