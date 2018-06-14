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

package webhook

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/blackrock/axis/common"
	"github.com/blackrock/axis/job"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var (
	client  = &http.Client{}
	payload = "{name: x}"
)

func createWebhookSignal(t *testing.T, httpMethod string, endpoint string) job.Signal {
	es := job.New(nil, nil, zap.NewNop())
	Webhook(es)
	webhookFactory, ok := es.GetCoreFactory(v1alpha1.SignalTypeWebhook)
	assert.True(t, ok, "webhook factory not found")
	abstractSignal := job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Webhook: &v1alpha1.WebhookSignal{
				Port:     common.WebhookServiceTargetPort,
				Endpoint: endpoint,
				Method:   httpMethod,
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}
	webhookSignal, err := webhookFactory.Create(abstractSignal)
	assert.Nil(t, err, "unable to create real webhook signal from abstract spec")
	return webhookSignal
}

func handleEvent(t *testing.T, testEventChan chan job.Event) {
	event := <-testEventChan
	assert.Equal(t, fmt.Sprintf("localhost:%d", common.WebhookServicePort), event.GetSource())
	assert.Equal(t, payload, string(event.GetBody()))
}

func makeAPIRequest(t *testing.T, httpMethod string, endpoint string) {
	webhookSignal := createWebhookSignal(t, httpMethod, endpoint)
	testEventChan := make(chan job.Event)
	webhookSignal.Start(testEventChan)

	go handleEvent(t, testEventChan)

	request, err := http.NewRequest(httpMethod, fmt.Sprintf("http://localhost:%d%s", common.WebhookServicePort, endpoint), strings.NewReader(payload))
	assert.Nil(t, err, "unable to create http request")
	request.Close = true // do not keep the connection alive
	resp, err := client.Do(request)
	assert.Nil(t, err, "failed to perform http request")
	assert.Equal(t, "200 OK", resp.Status)
	err = webhookSignal.Stop()
	assert.Nil(t, err, "failed to stop webhook signal")
}

func testPostRequest(t *testing.T) {
	makeAPIRequest(t, http.MethodPost, "/post")
}

func testPutRequest(t *testing.T) {
	makeAPIRequest(t, http.MethodPut, "/put")
}

func testDeleteRequest(t *testing.T) {
	makeAPIRequest(t, http.MethodDelete, "/delete")
}

func TestSignal(t *testing.T) {
	t.Run("post", testPostRequest)
	t.Run("put", testPutRequest)
	t.Run("delete", testDeleteRequest)
}
