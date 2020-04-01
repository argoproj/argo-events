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

	cd_v1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	gw_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	ss_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	rollouts_v1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	wf_v1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

// NOTE: custom resources must be manually added here
func init() {
	if err := wf_v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := ss_v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := gw_v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := rollouts_v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := cd_v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}

var (
	registry = runtime.NewEquivalentResourceRegistry()
)

// ArtifactReader enables reading artifacts from an external store
type ArtifactReader interface {
	Read() ([]byte, error)
}

// FetchArtifact from the location, decode it using explicit types, and unstructure it
func FetchArtifact(reader ArtifactReader, gvr *metav1.GroupVersionResource) (*unstructured.Unstructured, error) {
	var err error
	var obj []byte
	obj, err = reader.Read()
	if err != nil {
		return nil, err
	}
	return decodeAndUnstructure(obj, gvr)
}

// GetArtifactReader returns the ArtifactReader for this location
func GetArtifactReader(loc *ss_v1alpha1.ArtifactLocation, creds *Credentials, clientset kubernetes.Interface) (ArtifactReader, error) {
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
		return NewGitReader(clientset, loc.Git)
	}
	if loc.Configmap != nil {
		return NewConfigMapReader(clientset, loc.Configmap)
	}
	if loc.Resource != nil {
		return NewResourceReader(loc.Resource)
	}
	return nil, fmt.Errorf("unknown artifact location: %v", *loc)
}

func decodeAndUnstructure(b []byte, gvr *metav1.GroupVersionResource) (*unstructured.Unstructured, error) {
	gvk := registry.KindFor(schema.GroupVersionResource{
		Group:    gvr.Group,
		Version:  gvr.Version,
		Resource: gvr.Resource,
	}, "")

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(b, &gvk, nil)
	if err != nil {
		return nil, err
	}

	uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: uObj}, nil
}
