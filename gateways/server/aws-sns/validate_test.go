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

package aws_sns

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/gateways"
	esv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestSNSEventSourceExecutor_ValidateEventSource(t *testing.T) {
	listener := &EventListener{}

	valid, _ := listener.ValidateEventSource(context.Background(), &gateways.EventSource{
		Id:    "1",
		Name:  "sns",
		Value: nil,
		Type:  "sq",
	})
	assert.Equal(t, false, valid.IsValid)
	assert.Equal(t, "event source is not type of sns", valid.Reason)

	content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", gateways.EventSourceDir, "aws-sns.yaml"))
	assert.Nil(t, err)

	var eventSource *esv1alpha1.EventSource
	err = yaml.Unmarshal(content, &eventSource)
	assert.Nil(t, err)

	for name, value := range eventSource.Spec.SNS {
		fmt.Println(name)
		content, err := yaml.Marshal(value)
		assert.Nil(t, err)
		valid, _ := listener.ValidateEventSource(context.Background(), &gateways.EventSource{
			Id:    "1",
			Name:  "sns",
			Value: content,
			Type:  "sns",
		})
		fmt.Println(valid.Reason)
		assert.Equal(t, true, valid.IsValid)
	}
}
