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
	"net"
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
	client  = &http.Client{Timeout: 0}
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
	if err != nil {
		assert.Fail(t, "unable to create real webhook signal from abstract spec")
	}
	return webhookSignal
}

func handleEvent(t *testing.T, testEventChan chan job.Event) {
	event := <-testEventChan
	assert.Equal(t, event.GetSource(), fmt.Sprintf("localhost:%d", common.WebhookServicePort))
	body := event.GetBody()
	assert.NotNil(t, body)
	assert.Equal(t, string(body[:]), payload)
}

func makeAPIRequest(t *testing.T, httpMethod string, endpoint string) {
	webhookSignal := createWebhookSignal(t, httpMethod, endpoint)
	testEventChan := make(chan job.Event)
	webhookSignal.Start(testEventChan)

	go handleEvent(t, testEventChan)

	request, err := http.NewRequest(httpMethod, fmt.Sprintf("http://localhost:%d%s", common.WebhookServicePort, endpoint), strings.NewReader(payload))
	if err != nil {
		assert.Fail(t, "unable to create http request", err)
	}
	resp, err := client.Do(request)
	if err != nil && err.(net.Error).Timeout() {
		assert.Fail(t, "unable to connect to http server", err)
	}
	assert.Nil(t, err)
	assert.Equal(t, resp.Status, "200 OK")
	err = webhookSignal.Stop()
	assert.Equal(t, err, nil)
}

func TestSignal(t *testing.T) {
	makeAPIRequest(t, http.MethodPost, "/post")
	makeAPIRequest(t, http.MethodPut, "/put")
	makeAPIRequest(t, http.MethodDelete, "/delete")
}
