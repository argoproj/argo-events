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
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func (tw *testWeb) handleEvent(t *testing.T, testEventChan <-chan *v1alpha1.Event) {
	event := <-testEventChan

	if string(event.Data) != tw.payload {
		t.Errorf("event Data:\nexpected: %s\nactual: %s", tw.payload, string(event.Data))
	}
}

func (tw *testWeb) makeAPIRequest(t *testing.T, httpMethod string, endpoint string) {
	signal := v1alpha1.Signal{
		Name: "test",
		Webhook: &v1alpha1.WebhookSignal{
			Endpoint: endpoint,
			Method:   httpMethod,
		},
	}
	done := make(chan struct{})
	// stop listening and ensure the events channel is closed on exit
	defer close(done)
	events, err := tw.listener.Listen(&signal, done)

	go tw.handleEvent(t, events)

	request, err := http.NewRequest(httpMethod, fmt.Sprintf("http://localhost:%d%s", tw.port, endpoint), strings.NewReader(tw.payload))
	if err != nil {
		t.Fatalf("unable to create http request. cause: %s", err)
	}
	request.Close = true // do not keep the connection alive
	resp, err := tw.client.Do(request)
	if err != nil {
		t.Fatalf("failed to perform http request. cause: %s", err)
	}
	if resp.Status != "200 OK" {
		t.Errorf("response status expected: '200 OK' actual: '%s'", resp.Status)
	}
}

func (tw *testWeb) testPostRequest(t *testing.T) {
	tw.makeAPIRequest(t, http.MethodPost, "/post")
}

func (tw *testWeb) testPutRequest(t *testing.T) {
	tw.makeAPIRequest(t, http.MethodPut, "/put")
}

func (tw *testWeb) testDeleteRequest(t *testing.T) {
	tw.makeAPIRequest(t, http.MethodDelete, "/delete")
}

type testWeb struct {
	// we need to use the actual implementation to ultimately stop the server after tests are done
	port     int
	listener *webhook
	client   *http.Client
	payload  string
}

func TestSignal(t *testing.T) {
	tw := &testWeb{
		port:     5677,
		listener: New(5677).(*webhook),
		client:   &http.Client{},
		payload:  "{name: x}",
	}
	defer func() {
		tw.listener.srv.SetKeepAlivesEnabled(false)
		testCtx := context.TODO()
		err := tw.listener.srv.Shutdown(testCtx)
		if err != nil {
			t.Fatalf("failed to shutdown the http server: %s", err)
		}
	}()
	t.Run("post", tw.testPostRequest)
	t.Run("put", tw.testPutRequest)
	t.Run("delete", tw.testDeleteRequest)
}
