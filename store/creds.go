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

package store

import (
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Credentials contains the information necessary to access the artifact
type Credentials struct {
	accessKey string
	secretKey string
}

// GetCredentials for this artifact
func GetCredentials(kubeClient kubernetes.Interface, namespace string, art *v1alpha1.ArtifactLocation) (*Credentials, error) {
	if art.S3 != nil {
		accessKey, err := getSecrets(kubeClient, namespace, art.S3.AccessKey.Name, art.S3.AccessKey.Key)
		if err != nil {
			return nil, err
		}
		secretKey, err := getSecrets(kubeClient, namespace, art.S3.SecretKey.Name, art.S3.SecretKey.Key)
		if err != nil {
			return nil, err
		}
		return &Credentials{
			accessKey: accessKey,
			secretKey: secretKey,
		}, nil
	}
	// this should never happen
	return nil, fmt.Errorf("artifact is not an S3 artifact")
}

// getSecrets retrieves the secret value from the secret in namespace with name and key
func getSecrets(client kubernetes.Interface, namespace string, name, key string) (string, error) {
	secretsIf := client.CoreV1().Secrets(namespace)
	var secret *v1.Secret
	var err error
	_ = wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		secret, err = secretsIf.Get(name, metav1.GetOptions{})
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	val, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("secret '%s' does not have the key '%s'", name, key)
	}
	return string(val), nil
}
