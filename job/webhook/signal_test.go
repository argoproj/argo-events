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
	"testing"
	"github.com/blackrock/axis/job"
	"go.uber.org/zap"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/blackrock/axis/common"
	"net/http"
	"fmt"
	"strings"
	"net"
	"time"
)

var client = &http.Client{}

func createWebhookSignal(t *testing.T, httpMethod string, endpoint string) job.Signal {
	es := job.New(nil, nil, zap.NewNop())
	Webhook(es)
	webhookFactory, ok := es.GetFactory(v1alpha1.SignalTypeWebhook)
	assert.True(t, ok, "webhook factory not found")
	abstractSignal := job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Webhook: &v1alpha1.WebhookSignal{
				Port: common.WebhookServiceTargetPort,
				Endpoint: endpoint,
				Method: httpMethod,
			},
		},
		Log: zap.NewNop(),
		Session: es,
	}
	webhookSignal := webhookFactory.Create(abstractSignal)
	return webhookSignal
}

func handleEvent(t *testing.T, testEventChan chan job.Event) {
	event := <-testEventChan
	assert.Equal(t, event.GetSource(), fmt.Sprintf("localhost:%d", common.WebhookServicePort))
}

func makeAPIRequest(t *testing.T, httpMethod string, endpoint string) {
	webhookSignal := createWebhookSignal(t, httpMethod, endpoint)
	testEventChan := make(chan job.Event)
	webhookSignal.Start(testEventChan)

	go handleEvent(t, testEventChan)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf(":%d", common.WebhookServicePort), time.Second * 5)
	// Server has started
	if err == nil {
		conn.Close()
		request, err := http.NewRequest(httpMethod, fmt.Sprintf("http://localhost:%d%s", common.WebhookServicePort, endpoint), strings.NewReader("{name: x}"))
		if err != nil {
			assert.Fail(t, "unable to create http request", err)
		}
		resp, err := client.Do(request)
		assert.Nil(t, err)
		assert.Equal(t, resp.Status, "200 OK")
		err = webhookSignal.Stop()
		assert.Equal(t, err, nil)
	} else {
		assert.Fail(t, "unable to connect to http server")
	}
}

func TestSignal(t *testing.T) {
	makeAPIRequest(t, http.MethodPost, "/post")
	makeAPIRequest(t, http.MethodPut, "/put")
	makeAPIRequest(t, http.MethodDelete, "/delete")
}
