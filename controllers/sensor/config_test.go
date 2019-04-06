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

package sensor

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSensorControllerConfigWatch(t *testing.T) {
	sc := getSensorController()

	convey.Convey("Given a sensor", t, func() {
		convey.Convey("Create a new watch and make sure watcher is not nil", func() {
			watcher := sc.newControllerConfigMapWatch()
			convey.So(watcher, convey.ShouldNotBeNil)
		})
	})

	convey.Convey("Given a sensor, resync config", t, func() {
		convey.Convey("Update a sensor configmap with new instance id and remove namespace", func() {
			cmObj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: common.DefaultControllerNamespace,
					Name:      sc.ConfigMap,
				},
				Data: map[string]string{
					SensorControllerConfigMapKey: `instanceID: fake-instance-id`,
				},
			}
			cm, err := sc.kubeClientset.CoreV1().ConfigMaps(sc.Namespace).Create(cmObj)
			convey.Convey("Make sure no error occurs", func() {
				convey.So(err, convey.ShouldBeNil)

				convey.Convey("Updated sensor configmap must be non-nil", func() {
					convey.So(cm, convey.ShouldNotBeNil)

					convey.Convey("Resync the sensor configuration", func() {
						err := sc.ResyncConfig(cmObj.Namespace)
						convey.Convey("No error should occur while resyncing sensor configuration", func() {
							convey.So(err, convey.ShouldBeNil)

							convey.Convey("The updated instance id must be fake-instance-id", func() {
								convey.So(sc.Config.InstanceID, convey.ShouldEqual, "fake-instance-id")
								convey.So(sc.Config.Namespace, convey.ShouldBeEmpty)
							})
						})
					})
				})
			})
		})
	})
}
