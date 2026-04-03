package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
)

func TestHandleRoute(t *testing.T) {
	convey.Convey("Given a route that receives HTTP access", t, func() {
		router := &Router{
			route: webhook.GetFakeRoute(),
		}
		writer := &webhook.FakeHttpWriter{}

		convey.Convey("Test Get method with query parameters", func() {
			url, _ := url.Parse("http://example.com/fake?aaa=b%20b&ccc=d%20d")
			out := make(chan *webhook.Dispatch)
			router.route.Active = true
			router.route.Context.Method = http.MethodGet

			go func() {
				temp := <-router.route.DispatchChan
				temp.SuccessChan <- true
				out <- temp
			}()

			router.HandleRoute(writer, &http.Request{
				Method: http.MethodGet,
				URL:    url,
			})
			result := <-out
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			convey.So(string(result.Data), convey.ShouldContainSubstring, `"body":{"aaa":["b b"],"ccc":["d d"]}`)
		})
		convey.Convey("Test Get method without query parameter", func() {
			url, _ := url.Parse("http://example.com/fake")
			out := make(chan *webhook.Dispatch)
			router.route.Active = true
			router.route.Context.Method = http.MethodGet

			go func() {
				temp := <-router.route.DispatchChan
				temp.SuccessChan <- true
				out <- temp
			}()

			router.HandleRoute(writer, &http.Request{
				Method: http.MethodGet,
				URL:    url,
			})
			result := <-out
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			convey.So(string(result.Data), convey.ShouldContainSubstring, `"body":{}`)
		})
		convey.Convey("Test POST method with form-urlencoded", func() {
			payload := []byte(`aaa=b%20b&ccc=d%20d`)

			out := make(chan webhook.Dispatch)
			router.route.Active = true
			router.route.Context.Method = http.MethodPost

			go func() {
				temp := <-router.route.DispatchChan
				temp.SuccessChan <- true
				out <- *temp
			}()

			var buf bytes.Buffer
			buf.Write(payload)

			headers := make(map[string][]string)
			headers["Content-Type"] = []string{"application/x-www-form-urlencoded"}

			router.HandleRoute(writer, &http.Request{
				Method: http.MethodPost,
				Header: headers,
				Body:   io.NopCloser(strings.NewReader(buf.String())),
			})
			result := <-out
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			convey.So(string(result.Data), convey.ShouldContainSubstring, `"body":{"aaa":["b b"],"ccc":["d d"]}`)
		})
		convey.Convey("Test POST method with json", func() {
			payload := []byte(`{"aaa":["b b"],"ccc":["d d"]}`)

			out := make(chan webhook.Dispatch)
			router.route.Active = true
			router.route.Context.Method = http.MethodPost

			go func() {
				temp := <-router.route.DispatchChan
				temp.SuccessChan <- true
				out <- *temp
			}()

			var buf bytes.Buffer
			buf.Write(payload)

			headers := make(map[string][]string)
			headers["Content-Type"] = []string{"application/json"}

			router.HandleRoute(writer, &http.Request{
				Method: http.MethodPost,
				Header: headers,
				Body:   io.NopCloser(strings.NewReader(buf.String())),
			})
			result := <-out
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			convey.So(string(result.Data), convey.ShouldContainSubstring, `"body":{"aaa":["b b"],"ccc":["d d"]}`)
		})
	})
}

func TestParseIncomingCloudEvent(t *testing.T) {
	t.Run("detects CloudEvent binary content mode", func(t *testing.T) {
		body := json.RawMessage(`{"message":"hello"}`)
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/webhook", bytes.NewReader(body))
		req.Header.Set("Ce-Specversion", "1.0")
		req.Header.Set("Ce-Type", "com.example.test")
		req.Header.Set("Ce-Source", "https://example.com/source")
		req.Header.Set("Ce-Id", "my-custom-id")
		req.Header.Set("Content-Type", "application/json")

		result := parseIncomingCloudEvent(req, &body)

		require.NotNil(t, result)
		assert.Equal(t, "my-custom-id", result.ID())
		assert.Equal(t, "https://example.com/source", result.Source())
		assert.Equal(t, "com.example.test", result.Type())
	})

	t.Run("detects CloudEvent with extensions", func(t *testing.T) {
		body := json.RawMessage(`{"key":"value"}`)
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/webhook", bytes.NewReader(body))
		req.Header.Set("Ce-Specversion", "1.0")
		req.Header.Set("Ce-Type", "com.example.test")
		req.Header.Set("Ce-Source", "https://example.com/source")
		req.Header.Set("Ce-Id", "ext-test")
		req.Header.Set("Ce-Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		req.Header.Set("Content-Type", "application/json")

		result := parseIncomingCloudEvent(req, &body)

		require.NotNil(t, result)
		assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", result.Extensions()["traceparent"])
	})

	t.Run("returns nil for plain HTTP request", func(t *testing.T) {
		body := json.RawMessage(`{"message":"hello"}`)
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/webhook", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		result := parseIncomingCloudEvent(req, &body)

		assert.Nil(t, result)
	})

	t.Run("returns nil for nil body", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/webhook", nil)

		result := parseIncomingCloudEvent(req, nil)

		assert.Nil(t, result)
	})
}
