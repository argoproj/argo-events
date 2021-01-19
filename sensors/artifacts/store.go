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

package artifacts

import (
	"fmt"
	"strings"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ArtifactReader enables reading artifacts from an external store
type ArtifactReader interface {
	Read() ([]byte, error)
}

// FetchArtifact from the location, decode it using explicit types, and unstructure it
func FetchArtifact(reader ArtifactReader) (*unstructured.Unstructured, error) {
	var err error
	var obj []byte
	obj, err = reader.Read()
	if err != nil {
		return nil, err
	}
	return decodeAndUnstructure(obj)
}

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

// GetArtifactReader returns the ArtifactReader for this location
func GetArtifactReader(loc *v1alpha1.ArtifactLocation, creds *Credentials) (ArtifactReader, error) {
	if loc.S3 != nil {
		return NewS3Reader(loc.S3, creds)
	}
	if loc.Inline != nil {
		return NewInlineReader(loc.Inline)
	}
	if loc.File != nil {
		return NewFileReader(loc.File)
	}
	if loc.URL != nil {
		return NewURLReader(loc.URL)
	}
	if loc.Git != nil {
		return NewGitReader(loc.Git)
	}
	if loc.Configmap != nil {
		return NewConfigMapReader(loc.Configmap)
	}
	if loc.Resource != nil {
		return NewResourceReader(loc.Resource)
	}
	return nil, fmt.Errorf("unknown artifact location: %v", *loc)
}

func decodeAndUnstructure(b []byte) (*unstructured.Unstructured, error) {
	var result map[string]interface{}
	if err := yaml.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: result}, nil
}
