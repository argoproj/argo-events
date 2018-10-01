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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDefaultConfigMapName(t *testing.T) {
	res := DefaultConfigMapName("sensor-controller")
	assert.Equal(t, "sensor-controller-configmap", res)
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
