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

package gateway

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/smartystreets/goconvey/convey"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/informers"
)

func TestInformer(t *testing.T) {
	convey.Convey("Given a gateway controller", t, func() {
		controller := getGatewayController()
		convey.Convey("Instance ID required key must match", func() {
			req := controller.instanceIDReq()
			convey.So(req.Key(), convey.ShouldEqual, common.LabelKeyGatewayControllerInstanceID)
			convey.So(req.Operator(), convey.ShouldEqual, selection.Equals)
			convey.So(req.Values().Has("argo-events"), convey.ShouldBeTrue)
		})
	})

	convey.Convey("Given a gateway controller", t, func() {
		controller := getGatewayController()
		convey.Convey("Get a new gateway informer and make sure its not nil", func() {
			i := controller.newGatewayInformer()
			convey.So(i, convey.ShouldNotBeNil)
		})
	})

	convey.Convey("Given a gateway controller", t, func() {
		controller := getGatewayController()
		informerFactory := informers.NewSharedInformerFactory(controller.kubeClientset, 0)
		convey.Convey("Get a new gateway pod informer and make sure its not nil", func() {
			i := controller.newGatewayPodInformer(informerFactory)
			convey.So(i, convey.ShouldNotBeNil)
		})
	})

	convey.Convey("Given a gateway controller", t, func() {
		controller := getGatewayController()
		informerFactory := informers.NewSharedInformerFactory(controller.kubeClientset, 0)
		convey.Convey("Get a new gateway service informer and make sure its not nil", func() {
			i := controller.newGatewayServiceInformer(informerFactory)
			convey.So(i, convey.ShouldNotBeNil)
		})
	})
}
