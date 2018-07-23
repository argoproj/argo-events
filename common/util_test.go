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
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestRetryableKubeAPIError(t *testing.T) {
	errUnAuth := errors.NewUnauthorized("reason")
	errNotFound := errors.NewNotFound(v1alpha1.Resource("sensor"), "hello")
	errForbidden := errors.NewForbidden(v1alpha1.Resource("sensor"), "hello", nil)
	errInvalid := errors.NewInvalid(v1alpha1.Kind("core/data"), "hello", nil)
	errMethodNotSupported := errors.NewMethodNotSupported(v1alpha1.Resource("sensor"), "action")

	assert.True(t, IsRetryableKubeAPIError(errUnAuth))
	assert.False(t, IsRetryableKubeAPIError(errNotFound))
	assert.False(t, IsRetryableKubeAPIError(errForbidden))
	assert.False(t, IsRetryableKubeAPIError(errInvalid))
	assert.False(t, IsRetryableKubeAPIError(errMethodNotSupported))
}

func TestDefaultConfigMapName(t *testing.T) {
	res := DefaultConfigMapName("controller")
	assert.Equal(t, "controller-configmap", res)
}

func TestServerResourceForGroupVersionKind(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fakeDisco := fakeClient.Discovery()
	gvk := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}
	apiResource, err := ServerResourceForGroupVersionKind(fakeDisco, gvk)
	fmt.Println(err)
	assert.NotNil(t, err)
	assert.Nil(t, apiResource)
}

func TestResolveNamespace(t *testing.T) {
	defer os.Unsetenv(EnvVarNamespace)

	RefreshNamespace()
	assert.Equal(t, "default", DefaultSensorControllerNamespace)

	// TODO: now write the namespace file

	// now set the env variable
	err := os.Setenv(EnvVarNamespace, "test")
	if err != nil {
		t.Error(err)
	}

	RefreshNamespace()
	assert.Equal(t, "test", DefaultSensorControllerNamespace)
}
