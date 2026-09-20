package webhook

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
)

func TestFormPayloadRespectsDefaultMaxPayloadSize(t *testing.T) {
	route := webhook.GetFakeRoute()
	context := *route.Context
	route.Context = &context
	route.Context.Method = http.MethodPost
	route.Active = true
	router := &Router{route: route}

	server := httptest.NewServer(http.HandlerFunc(router.HandleRoute))
	defer server.Close()
	dispatched := make(chan bool, 1)
	done := make(chan struct{})
	go func() {
		select {
		case event := <-route.DispatchChan:
			dispatched <- true
			event.SuccessChan <- true
		case <-done:
		}
	}()
	form := "a=" + strings.Repeat("x", int(v1alpha1.DefaultMaxWebhookPayloadSize))
	response, err := http.Post(server.URL+"/fake", "application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		close(done)
		t.Fatal(err)
	}
	defer response.Body.Close()
	close(done)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
	if !strings.Contains(string(body), "http: request body too large") {
		t.Errorf("response = %q, want request body too large error", body)
	}
	select {
	case <-dispatched:
		t.Error("oversized form was dispatched")
	default:
	}
}
