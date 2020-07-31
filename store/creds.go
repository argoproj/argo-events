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
	"strings"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// Credentials contains the information necessary to access the minio
type Credentials struct {
	accessKey string
	secretKey string
}

// GetCredentials for this minio
func GetCredentials(art *v1alpha1.ArtifactLocation) (*Credentials, error) {
	if art.S3 != nil {
		accessKey, err := common.GetSecretFromVolume(art.S3.AccessKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve accessKey")
		}
		secretKey, err := common.GetSecretFromVolume(art.S3.SecretKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve secretKey")
		}
		return &Credentials{
			accessKey: strings.TrimSpace(accessKey),
			secretKey: strings.TrimSpace(secretKey),
		}, nil
	}

	return nil, nil
}
