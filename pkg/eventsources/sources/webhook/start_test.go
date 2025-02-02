package webhook

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/smartystreets/goconvey/convey"

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
