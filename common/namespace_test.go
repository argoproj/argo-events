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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveNamespace(t *testing.T) {
	defer os.Unsetenv(EnvVarControllerNamespace)

	assert.Equal(t, "argo-events", DefaultControllerNamespace)

	// TODO: now write the namespace file

	// now set the env variable
	err := os.Setenv(EnvVarControllerNamespace, "test")
	if err != nil {
		t.Error(err)
	}

	RefreshNamespace()
	assert.Equal(t, "test", DefaultControllerNamespace)
}
