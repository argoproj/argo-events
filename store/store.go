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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"

	// import packages for the universal deserializer
	gw_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	ss_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	wf_v1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// NOTE: custom resources must be manually added here
func init() {
	wf_v1alpha1.AddToScheme(scheme.Scheme)
	ss_v1alpha1.AddToScheme(scheme.Scheme)
	gw_v1alpha1.AddToScheme(scheme.Scheme)
}

// ArtifactReader enables reading artifacts from an external store
type ArtifactReader interface {
	Read() ([]byte, error)
}

// FetchArtifact from the location, decode it using explicit types, and unstructure it
func FetchArtifact(reader ArtifactReader, gvk ss_v1alpha1.GroupVersionKind) (*unstructured.Unstructured, error) {
	var err error
	var obj []byte
	obj, err = reader.Read()
	if err != nil {
		return nil, err
	}
	return decodeAndUnstructure(obj, gvk)
}

// GetArtifactReader returns the ArtifactReader for this location
func GetArtifactReader(loc *ss_v1alpha1.ArtifactLocation, creds *Credentials) (ArtifactReader, error) {
	if loc.S3 != nil {
		return NewS3Reader(loc.S3, creds)
	} else if loc.Inline != nil {
		return NewInlineReader(loc.Inline)
	} else if loc.File != nil {
		return NewFileReader(loc.File)
	} else if loc.URL != nil {
		return NewURLReader(loc.URL)
	}
	return nil, fmt.Errorf("unknown artifact location: %v", *loc)
}

func decodeAndUnstructure(b []byte, gvk ss_v1alpha1.GroupVersionKind) (*unstructured.Unstructured, error) {
	gvk1 := &schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(b, gvk1, nil)
	if err != nil {
		return nil, err
	}

	uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: uObj}, nil
}
