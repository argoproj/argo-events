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

//
//import (
//	"encoding/json"
//	"testing"
//
//	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
//	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
//	"github.com/smartystreets/goconvey/convey"
//	corev1 "k8s.io/api/core/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
//	"k8s.io/apimachinery/pkg/labels"
//	"k8s.io/apimachinery/pkg/runtime"
//	"k8s.io/apimachinery/pkg/runtime/schema"
//	"k8s.io/apimachinery/pkg/types"
//	"k8s.io/apimachinery/pkg/watch"
//	"k8s.io/client-go/dynamic"
//	dynamicfake "k8s.io/client-go/dynamic/fake"
//	"k8s.io/client-go/kubernetes/fake"
//	kTesting "k8s.io/client-go/testing"
//	"k8s.io/client-go/util/flowcontrol"
//)
//
//var successLabels = map[string]string{
//	"success-label": "fake",
//}
//
//var failureLabels = map[string]string{
//	"failure-label": "fake",
//}
//
//var podTemplate = &corev1.Pod{
//	TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
//	Spec: corev1.PodSpec{
//		Containers: []corev1.Container{
//			{
//				Name:  "test1",
//				Image: "docker/whalesay",
//			},
//		},
//	},
//}
//
//var triggerTemplate = v1alpha1.Trigger{
//	Template: &v1alpha1.TriggerTemplate{
//		GroupVersionResource: &metav1.GroupVersionResource{
//			Resource:    "pods",
//			Version: "v1",
//		},
//	},
//}
//
//func getUnstructured(res interface{}) (*unstructured.Unstructured, error) {
//	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(res)
//	if err != nil {
//		return nil, err
//	}
//	return &unstructured.Unstructured{Object: obj}, nil
//}
//
//func TestProcessTrigger(t *testing.T) {
//	convey.Convey("Given a sensor", t, func() {
//		trigger := *triggerTemplate.DeepCopy()
//		trigger.Template.Name = "testTrigger"
//		pod := podTemplate.DeepCopy()
//		pod.Name = "testTrigger"
//		uObj, err := getUnstructured(pod)
//		convey.So(err, convey.ShouldBeNil)
//		trigger.Template.Source = &v1alpha1.ArtifactLocation{
//			Resource: uObj,
//		}
//		testSensor, err := getSensor()
//		convey.So(err, convey.ShouldBeNil)
//		soc := getsensorExecutionCtx(testSensor)
//		err = soc.executeTrigger(trigger)
//		convey.So(err, convey.ShouldBeNil)
//	})
//}
//
//type FakeName struct {
//	First string `json:"first"`
//	Last  string `json:"last"`
//}
//
//type fakeEvent struct {
//	Name         string `json:"name"`
//	Namespace    string `json:"namespace"`
//	Group        string `json:"group"`
//	GenerateName string `json:"generateName"`
//	Kind         string `json:"kind"`
//}
//
//func TestTriggerParameterization(t *testing.T) {
//	convey.Convey("Given an event, parameterize the trigger", t, func() {
//		testSensor, err := getSensor()
//		convey.So(err, convey.ShouldBeNil)
//		soc := getsensorExecutionCtx(testSensor)
//		triggerName := "test-workflow-trigger"
//		dependency := "test-gateway:test"
//
//		fe := &fakeEvent{
//			Namespace:    "fake-namespace",
//			Name:         "fake",
//			Group:        "v1",
//			GenerateName: "fake-",
//			Kind:         "Deployment",
//		}
//		eventBytes, err := json.Marshal(fe)
//		convey.So(err, convey.ShouldBeNil)
//
//		node := v1alpha1.NodeStatus{
//			Event: &apicommon.Event{
//				Payload: eventBytes,
//				Context: apicommon.EventContext{
//					Source: &apicommon.URI{
//						Host: dependency,
//					},
//					ContentType: "application/json",
//				},
//			},
//			Name:  dependency,
//			Type:  v1alpha1.NodeTypeEventDependency,
//			ID:    "1234",
//			Phase: v1alpha1.NodePhaseActive,
//		}
//
//		trigger := triggerTemplate.DeepCopy()
//		trigger.Template.Name = triggerName
//
//		trigger.TemplateParameters = []v1alpha1.TriggerParameter{
//			{
//				Src: &v1alpha1.TriggerParameterSource{
//					Event: dependency,
//					Path:  "name",
//				},
//				Dest: "name",
//			},
//		}
//
//		trigger.ResourceParameters = []v1alpha1.TriggerParameter{
//			{
//				Src: &v1alpha1.TriggerParameterSource{
//					Event: dependency,
//					Path:  "name",
//				},
//				Dest: "metadata.generateName",
//			},
//		}
//
//		nodeId := soc.sensor.NodeID(dependency)
//		wfNodeId := soc.sensor.NodeID(triggerName)
//
//		wfnode := v1alpha1.NodeStatus{
//			Event: &apicommon.Event{
//				Payload: eventBytes,
//				Context: apicommon.EventContext{
//					Source: &apicommon.URI{
//						Host: dependency,
//					},
//					ContentType: "application/json",
//				},
//			},
//			Name:  triggerName,
//			Type:  v1alpha1.NodeTypeTrigger,
//			ID:    "1234",
//			Phase: v1alpha1.NodePhaseNew,
//		}
//
//		soc.sensor.Status.Nodes = map[string]v1alpha1.NodeStatus{
//			nodeId:   node,
//			wfNodeId: wfnode,
//		}
//
//		err = soc.applyParamsTrigger(trigger)
//		convey.So(err, convey.ShouldBeNil)
//		convey.So(trigger.Template.Name, convey.ShouldEqual, fe.Name)
//
//		rObj := podTemplate.DeepCopy()
//		rObj.Name = "testTrigger"
//		uObj, err := getUnstructured(rObj)
//		convey.So(err, convey.ShouldBeNil)
//
//		err = soc.applyParamsResource(trigger.ResourceParameters, uObj)
//		convey.So(err, convey.ShouldBeNil)
//
//	})
//}
//
//func TestTriggerPolicy(t *testing.T) {
//	convey.Convey("Given a trigger, apply policy", t, func() {
//		testSensor, err := getSensor()
//		convey.So(err, convey.ShouldBeNil)
//		soc := getsensorExecutionCtx(testSensor)
//
//		trigger1 := triggerTemplate.DeepCopy()
//		trigger2 := triggerTemplate.DeepCopy()
//
//		trigger1.Template.Name = "testTrigger1"
//		trigger2.Template.Name = "testTrigger2"
//
//		triggerPod1 := podTemplate.DeepCopy()
//		triggerPod2 := podTemplate.DeepCopy()
//
//		triggerPod1.Name = "testPod1"
//		triggerPod2.Name = "testPod2"
//
//		triggerPod1.Labels = successLabels
//		triggerPod2.Labels = failureLabels
//
//		uObj1, err := getUnstructured(triggerPod1)
//		convey.So(err, convey.ShouldBeNil)
//
//		uObj2, err := getUnstructured(triggerPod2)
//		convey.So(err, convey.ShouldBeNil)
//
//		backoff := v1alpha1.Backoff{
//			Duration: 1000000000,
//			Factor:   2,
//			Steps:    10,
//		}
//
//		trigger1.Template.Source = &v1alpha1.ArtifactLocation{
//			Resource: uObj1,
//		}
//		trigger1.Policy = &v1alpha1.TriggerPolicy{
//			Backoff: backoff,
//			State: &v1alpha1.TriggerStateLabels{
//				Success: successLabels,
//			},
//		}
//
//		trigger2.Template.Source = &v1alpha1.ArtifactLocation{
//			Resource: uObj2,
//		}
//		trigger2.Policy = &v1alpha1.TriggerPolicy{
//			Backoff: backoff,
//			State: &v1alpha1.TriggerStateLabels{
//				Failure: failureLabels,
//			},
//		}
//
//		convey.Convey("Execute the first trigger  and make sure the trigger execution results in success", func() {
//			err = soc.executeTrigger(*trigger1)
//			convey.So(err, convey.ShouldBeNil)
//		})
//
//		convey.Convey("Execute the second trigger and make sure the trigger execution results in failure", func() {
//			err = soc.executeTrigger(*trigger2)
//			convey.So(err, convey.ShouldNotBeNil)
//		})
//
//		// modify backoff so that applyPolicy doesnt wait too much
//		trigger1.Policy.Backoff = v1alpha1.Backoff{
//			Steps:    2,
//			Duration: 1000000000,
//			Factor:   1,
//		}
//
//		triggerPod1.Labels = nil
//		uObj1, err = getUnstructured(triggerPod1)
//		convey.So(err, convey.ShouldBeNil)
//		trigger1.Template.Source.Resource = uObj1
//
//		convey.Convey("If trigger times out and error on timeout is set, trigger execution must fail", func() {
//			trigger1.Policy.ErrorOnBackoffTimeout = true
//			err = soc.executeTrigger(*trigger1)
//			convey.So(err, convey.ShouldNotBeNil)
//		})
//
//		convey.Convey("If trigger times out and error on timeout is not set, trigger execution must succeed", func() {
//			trigger1.Policy.ErrorOnBackoffTimeout = false
//			err = soc.executeTrigger(*trigger1)
//			convey.So(err, convey.ShouldBeNil)
//		})
//	})
//}
//
//func TestCreateResourceObject(t *testing.T) {
//	convey.Convey("Given a trigger", t, func() {
//		testSensor, err := getSensor()
//		convey.So(err, convey.ShouldBeNil)
//		soc := getsensorExecutionCtx(testSensor)
//		fakeclient := soc.dynamicClient.(*FakeClientPool).Fake
//		dynamicClient := dynamicfake.FakeResourceClient{Resource: schema.GroupVersionResource{Version: "v1", Resource: "pods"}, Fake: &fakeclient}
//
//		convey.Convey("Given a pod spec, create a pod trigger", func() {
//			pod := podTemplate.DeepCopy()
//			pod.Name = "testTrigger"
//			pod.Namespace = "foo"
//			uObj, err := getUnstructured(pod)
//			convey.So(err, convey.ShouldBeNil)
//
//			trigger := triggerTemplate.DeepCopy()
//			trigger.Template.Name = "trigger"
//
//			trigger.Template.Source = &v1alpha1.ArtifactLocation{
//				Resource: uObj,
//			}
//
//			convey.Println(trigger.Template.Source)
//
//			err = soc.createResourceObject(trigger, uObj)
//			convey.So(err, convey.ShouldBeNil)
//
//			unstructuredPod, err := dynamicClient.Get(pod.Name, metav1.GetOptions{})
//			convey.So(err, convey.ShouldBeNil)
//			convey.So(unstructuredPod.GetNamespace(), convey.ShouldEqual, "foo")
//		})
//
//		convey.Convey("Given a pod without namespace,create a pod trigger", func() {
//			pod := podTemplate.DeepCopy()
//			pod.Name = "testTrigger"
//			uObj, err := getUnstructured(pod)
//			convey.So(err, convey.ShouldBeNil)
//
//			trigger := triggerTemplate.DeepCopy()
//			trigger.Template.Name = "trigger"
//
//			trigger.Template.Source = &v1alpha1.ArtifactLocation{
//				Resource: uObj,
//			}
//
//			err = soc.createResourceObject(trigger, uObj)
//			convey.So(err, convey.ShouldBeNil)
//
//			unstructuredPod, err := dynamicClient.Get(pod.Name, metav1.GetOptions{})
//			convey.So(err, convey.ShouldBeNil)
//			convey.So(unstructuredPod.GetNamespace(), convey.ShouldEqual, testSensor.Namespace)
//		})
//	})
//}
//
//func TestExtractEvents(t *testing.T) {
//	convey.Convey("Given a sensor, extract events", t, func() {
//		sensor, _ := getSensor()
//		sec := getsensorExecutionCtx(sensor)
//		id := sensor.NodeID("test-gateway:test")
//		sensor.Status.Nodes = map[string]v1alpha1.NodeStatus{
//			id: {
//				Type: v1alpha1.NodeTypeEventDependency,
//				Event: &apicommon.Event{
//					Payload: []byte("hello"),
//					Context: apicommon.EventContext{
//						Source: &apicommon.URI{
//							Host: "test-gateway:test",
//						},
//					},
//				},
//			},
//		}
//		extractedEvents := sec.extractEvents([]v1alpha1.TriggerParameter{
//			{
//				Src: &v1alpha1.TriggerParameterSource{
//					Event: "test-gateway:test",
//				},
//				Dest: "fake-dest",
//			},
//		})
//		convey.So(len(extractedEvents), convey.ShouldEqual, 1)
//	})
//}
//
//func TestCanProcessTriggers(t *testing.T) {
//	convey.Convey("Given a sensor, test if triggers can be processed", t, func() {
//		sensor, err := getSensor()
//		convey.So(err, convey.ShouldBeNil)
//
//		sensor.Status.Nodes = map[string]v1alpha1.NodeStatus{
//			sensor.NodeID(sensor.Spec.Dependencies[0].Name): {
//				Name:  sensor.Spec.Dependencies[0].Name,
//				Phase: v1alpha1.NodePhaseComplete,
//				Type:  v1alpha1.NodeTypeEventDependency,
//			},
//		}
//
//		for _, dep := range []v1alpha1.EventDependency{
//			{
//				Name: "test-gateway:test2",
//			},
//			{
//				Name: "test-gateway:test3",
//			},
//		} {
//			sensor.Spec.Dependencies = append(sensor.Spec.Dependencies, dep)
//			sensor.Status.Nodes[sensor.NodeID(dep.Name)] = v1alpha1.NodeStatus{
//				Name:  dep.Name,
//				Phase: v1alpha1.NodePhaseComplete,
//				Type:  v1alpha1.NodeTypeEventDependency,
//			}
//		}
//
//		soc := getsensorExecutionCtx(sensor)
//		ok, err := soc.canProcessTriggers()
//		convey.So(err, convey.ShouldBeNil)
//		convey.So(ok, convey.ShouldEqual, true)
//
//		node := sensor.Status.Nodes[sensor.NodeID("test-gateway:test2")]
//		node.Phase = v1alpha1.NodePhaseNew
//		sensor.Status.Nodes[sensor.NodeID("test-gateway:test2")] = node
//
//		ok, err = soc.canProcessTriggers()
//		convey.So(err, convey.ShouldBeNil)
//		convey.So(ok, convey.ShouldEqual, false)
//
//		convey.Convey("Add dependency groups and evaluate the circuit", func() {
//			for _, depGroup := range []v1alpha1.DependencyGroup{
//				{
//					Name:         "depg1",
//					Dependencies: []string{sensor.Spec.Dependencies[1].Name, sensor.Spec.Dependencies[2].Name},
//				},
//				{
//					Name:         "depg2",
//					Dependencies: []string{sensor.Spec.Dependencies[0].Name},
//				},
//			} {
//				sensor.Spec.DependencyGroups = append(sensor.Spec.DependencyGroups, depGroup)
//				sensor.Status.Nodes[sensor.NodeID(depGroup.Name)] = v1alpha1.NodeStatus{
//					Name:  depGroup.Name,
//					Phase: v1alpha1.NodePhaseNew,
//				}
//			}
//
//			sensor.Spec.Circuit = "depg1 || depg2"
//
//			ok, err = soc.canProcessTriggers()
//			convey.So(err, convey.ShouldBeNil)
//			convey.So(ok, convey.ShouldEqual, true)
//		})
//
//		convey.Convey("If the previous round of triggers failed and error on previous round policy is set, then don't execute the triggers", func() {
//			sensor.Spec.ErrorOnFailedRound = true
//			sensor.Status.TriggerCycleStatus = v1alpha1.TriggerCycleFailure
//
//			ok, err = soc.canProcessTriggers()
//			convey.So(err, convey.ShouldNotBeNil)
//			convey.So(ok, convey.ShouldEqual, false)
//		})
//	})
//}
