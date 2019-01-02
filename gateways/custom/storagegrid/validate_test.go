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

package storagegrid

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
endpoint: "/"
port: "8080"
events:
    - "ObjectCreated:Put"
filter:
    suffix: ".txt"
    prefix: "hello-"
`
)

func TestStorageGridConfigExecutor_Validate(t *testing.T) {
	s3Config := &StorageGridConfigExecutor{}
	ctx := &gateways.EventSourceContext{
		Data: &gateways.EventSourceData{},
	}
	ctx.Data.Config = configValue
	err := s3Config.Validate(ctx)
	assert.Nil(t, err)

	badConfig := `
events:
    - "ObjectCreated:Put"
filter:
    suffix: ".txt"
    prefix: "hello-"
`

	ctx.Data.Config = badConfig

	err = s3Config.Validate(ctx)
	assert.NotNil(t, err)
}
