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

package artifact

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/argoproj/argo-events/gateways"
)

func TestS3ConfigExecutor_StopConfig(t *testing.T) {
	s3Config := &S3ConfigExecutor{}
	ctx := gateways.GetDefaultConfigContext(configKey)
	ctx.Active = true
	go func() {
		msg :=<- ctx.StopCh
		assert.Equal(t, msg, struct {}{})
	}()
	err := s3Config.StopConfig(ctx)
	assert.Nil(t, err)
}
