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
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateEventSource(t *testing.T) {
	convey.Convey("Given a valid webhook event source spec, parse it and make sure no error occurs", t, func() {
		ese := &WebhookEventSourceExecutor{
			Log: common.NewArgoEventsLogger(),
		}
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", gwcommon.EventSourceDir, "webhook.yaml"))
		convey.So(err, convey.ShouldBeNil)

		var cm *corev1.ConfigMap
		err = yaml.Unmarshal(content, &cm)
		convey.So(err, convey.ShouldBeNil)
		convey.So(cm, convey.ShouldNotBeNil)

		err = common.CheckEventSourceVersion(cm)
		convey.So(err, convey.ShouldBeNil)

		for key, value := range cm.Data {
			valid, _ := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
				Name:    key,
				Id:      common.Hasher(key),
				Data:    value,
				Version: cm.Labels[common.LabelArgoEventsEventSourceVersion],
			})
			convey.So(valid, convey.ShouldNotBeNil)
			convey.So(valid.IsValid, convey.ShouldBeTrue)
		}
	})
}

func TestValidate(t *testing.T) {
	convey.Convey("Given a webhook, validate it", t, func() {
		w := &gwcommon.Webhook{
			Port:     "12000",
			Endpoint: "/",
			Method:   "POST",
		}
		err := validateWebhook(w)
		convey.So(err, convey.ShouldBeNil)
	})
}
