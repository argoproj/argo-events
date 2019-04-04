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

package sensors

import (
	"encoding/json"
	"github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/ghodss/yaml"
	"testing"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

var testPod = `
apiVersion: v1
kind: Pod
metadata:
  generateName: test-
spec:
  containers:
  - name: test
    image: docker/whalesay
`

var testTrigger = v1alpha1.Trigger{
	Template: &v1alpha1.TriggerTemplate{
		Name: "sample",
		GroupVersionKind: &metav1.GroupVersionKind{
			Kind:    "Pod",
			Version: "v1",
		},
		Source: &v1alpha1.ArtifactLocation{
			Inline: &testPod,
		},
	},
}

func TestProcessTrigger(t *testing.T) {
	convey.Convey("Given a sensor", t, func() {
		triggers := make([]v1alpha1.Trigger, 1)
		triggers[0] = testTrigger
		testSensor, err := getSensor()
		convey.So(err, convey.ShouldBeNil)
		testSensor.Spec.Triggers = triggers
		soc := getsensorExecutionCtx(testSensor)
		err = soc.executeTrigger(testTrigger)
		convey.So(err, convey.ShouldBeNil)
	})
}

type FakeName struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

type fakeEvent struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Group        string `json:"group"`
	GenerateName string `json:"generateName"`
	Kind         string `json:"kind"`
}

func TestCreateResourceObject(t *testing.T) {
	convey.Convey("Given a resource object", t, func() {
		testSensor, err := getSensor()
		convey.So(err, convey.ShouldBeNil)
		soc := getsensorExecutionCtx(testSensor)
		fakeclient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

		convey.Convey("Given a pod spec, get a pod object", func() {
			pod := &corev1.Pod{
				TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "my-pod"},
			}
			uObj, err := getUnstructured(pod)
			convey.So(err, convey.ShouldBeNil)

			err = soc.createResourceObject(&testTrigger, uObj)
			convey.So(err, convey.ShouldBeNil)

			unstructuredPod, err := fakeclient.Resource(schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			}).Get(pod.Name, metav1.GetOptions{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(unstructuredPod.GetNamespace(), convey.ShouldEqual, "foo")
		})

		fe := &fakeEvent{
			Namespace:    "fake-namespace",
			Name:         "fake",
			Group:        "v1",
			GenerateName: "fake-",
			Kind:         "Deployment",
		}
		eventBytes, err := json.Marshal(fe)
		convey.So(err, convey.ShouldBeNil)

		node := v1alpha1.NodeStatus{
			Event: &common.Event{
				Payload: eventBytes,
				Context: common.EventContext{
					Source: &common.URI{
						Host: "test-gateway:test",
					},
					ContentType: "application/json",
				},
			},
			Name:  "test-gateway:test",
			Type:  v1alpha1.NodeTypeEventDependency,
			ID:    "1234",
			Phase: v1alpha1.NodePhaseActive,
		}

		testTrigger.TemplateParameters = []v1alpha1.TriggerParameter{
			{
				Src: &v1alpha1.TriggerParameterSource{
					Event: "test-gateway:test",
					Path:  "name",
				},
				Dest: "name",
			},
		}

		testTrigger.ResourceParameters = []v1alpha1.TriggerParameter{
			{
				Src: &v1alpha1.TriggerParameterSource{
					Event: "test-gateway:test",
					Path:  "name",
				},
				Dest: "metadata.generateName",
			},
		}

		nodeId := soc.sensor.NodeID("test-gateway:test")
		wfNodeId := soc.sensor.NodeID("test-workflow-trigger")

		wfnode := v1alpha1.NodeStatus{
			Event: &common.Event{
				Payload: eventBytes,
				Context: common.EventContext{
					Source: &common.URI{
						Host: "test-gateway:test",
					},
					ContentType: "application/json",
				},
			},
			Name:  "test-workflow-trigger",
			Type:  v1alpha1.NodeTypeTrigger,
			ID:    "1234",
			Phase: v1alpha1.NodePhaseNew,
		}

		soc.sensor.Status.Nodes = map[string]v1alpha1.NodeStatus{
			nodeId:   node,
			wfNodeId: wfnode,
		}

		convey.Convey("Given parameters for trigger template, apply params", func() {
			err = soc.applyParamsTrigger(&testTrigger)
			convey.So(err, convey.ShouldBeNil)
			convey.So(testTrigger.Template.Name, convey.ShouldEqual, fe.Name)

			var tp corev1.Pod
			err = yaml.Unmarshal([]byte(testPod), &tp)
			convey.So(err, convey.ShouldBeNil)

			rObj := tp.DeepCopy()
			uObj, err := getUnstructured(rObj)
			convey.So(err, convey.ShouldBeNil)

			err = soc.applyParamsResource(testTrigger.ResourceParameters, uObj)
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("Given a pod without namespace, use sensor namespace", func() {
			pod := &corev1.Pod{
				TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "my-pod-without-namespace"},
			}
			uObj, err := getUnstructured(pod)
			convey.So(err, convey.ShouldBeNil)

			err = soc.createResourceObject(&testTrigger, uObj)
			convey.So(err, convey.ShouldBeNil)

			unstructuredPod, err := fakeclient.Resource(schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			}).Get(pod.Name, metav1.GetOptions{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(unstructuredPod.GetNamespace(), convey.ShouldEqual, testSensor.Namespace)
		})
	})
}

func getUnstructured(res interface{}) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(res)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

func TestExtractEvents(t *testing.T) {
	convey.Convey("Given a sensor, extract events", t, func() {
		sensor, _ := getSensor()
		sec := getsensorExecutionCtx(sensor)
		id := sensor.NodeID("test-gateway:test")
		sensor.Status.Nodes = map[string]v1alpha1.NodeStatus{
			id: {
				Type: v1alpha1.NodeTypeEventDependency,
				Event: &apicommon.Event{
					Payload: []byte("hello"),
					Context: apicommon.EventContext{
						Source: &apicommon.URI{
							Host: "test-gateway:test",
						},
					},
				},
			},
		}
		extractedEvents := sec.extractEvents([]v1alpha1.TriggerParameter{
			{
				Src: &v1alpha1.TriggerParameterSource{
					Event: "test-gateway:test",
				},
				Dest: "fake-dest",
			},
		})
		convey.So(len(extractedEvents), convey.ShouldEqual, 1)
	})
}
