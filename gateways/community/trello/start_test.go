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

package trello

import (
	"bytes"
	"github.com/argoproj/argo-events/common"
	"io/ioutil"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"testing"

	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		rc := gwcommon.GetFakeRoute()
		helper.ActiveEndpoints[rc.Webhook.Endpoint] = &gwcommon.Endpoint{
			DataCh: make(chan []byte),
		}

		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		rc.Configs[LabelTrelloConfig] = ps.(*trello)
		writer := &gwcommon.FakeHttpWriter{}

		convey.Convey("Inactive route should return error", func() {
			pbytes, err := yaml.Marshal(ps.(*trello))
			convey.So(err, convey.ShouldBeNil)
			RouteActiveHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(pbytes)),
			}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("Active route should return success", func() {
			helper.ActiveEndpoints[rc.Webhook.Endpoint].Active = true
			dataCh := make(chan []byte)
			go func() {
				resp := <-helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh
				dataCh <- resp
			}()

			RouteActiveHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader([]byte("fake notification"))),
			}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			data := <-dataCh
			convey.So(string(data), convey.ShouldEqual, "fake notification")
		})

		convey.Convey("Run post activate", func() {
			ese := &TrelloEventSourceExecutor{
				Clientset: fake.NewSimpleClientset(),
				Log:       common.GetLoggerContext(common.LoggerConf()).Logger(),
				Namespace: "fake",
			}
			secret, err := ese.Clientset.CoreV1().Secrets(ese.Namespace).Create(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "trello",
					Namespace: ese.Namespace,
				},
				Data: map[string][]byte{
					"api":   []byte("api"),
					"token": []byte("token"),
				},
			})
			convey.So(err, convey.ShouldBeNil)
			convey.So(secret, convey.ShouldNotBeNil)

			err = ese.PostActivate(rc)
			convey.So(err, convey.ShouldNotBeNil)
		})

	})
}
