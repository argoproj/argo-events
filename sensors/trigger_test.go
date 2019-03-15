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

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	kTesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/flowcontrol"
)

// Below code refers to PR https://github.com/kubernetes/kubernetes/issues/60390

// FakeClient is a fake implementation of dynamic.Interface.
type FakeClient struct {
	GroupVersion schema.GroupVersion

	*kTesting.Fake
}

// GetRateLimiter returns the rate limiter for this client.
func (c *FakeClient) GetRateLimiter() flowcontrol.RateLimiter {
	return nil
}

// Resource returns an API interface to the specified resource for this client's
// group and version.  If resource is not a namespaced resource, then namespace
// is ignored.  The ResourceClient inherits the parameter codec of this client
func (c *FakeClient) Resource(resource *metav1.APIResource, namespace string) dynamic.ResourceInterface {
	return &FakeResourceClient{
		Resource:  c.GroupVersion.WithResource(resource.Name),
		Kind:      c.GroupVersion.WithKind(resource.Kind),
		Namespace: namespace,

		Fake: c.Fake,
	}
}

// ParameterCodec returns a client with the provided parameter codec.
func (c *FakeClient) ParameterCodec(parameterCodec runtime.ParameterCodec) dynamic.Interface {
	return &FakeClient{
		Fake: c.Fake,
	}
}

// FakeResourceClient is a fake implementation of dynamic.ResourceInterface
type FakeResourceClient struct {
	Resource  schema.GroupVersionResource
	Kind      schema.GroupVersionKind
	Namespace string

	*kTesting.Fake
}

// List returns a list of objects for this resource.
func (c *FakeResourceClient) List(opts metav1.ListOptions) (runtime.Object, error) {
	obj, err := c.Fake.
		Invokes(kTesting.NewListAction(c.Resource, c.Kind, c.Namespace, opts), &unstructured.UnstructuredList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := kTesting.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &unstructured.UnstructuredList{}
	for _, item := range obj.(*unstructured.UnstructuredList).Items {
		if label.Matches(labels.Set(item.GetLabels())) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Get gets the resource with the specified name.
func (c *FakeResourceClient) Get(name string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	obj, err := c.Fake.
		Invokes(kTesting.NewGetAction(c.Resource, c.Namespace, name), &unstructured.Unstructured{})

	if obj == nil {
		return nil, err
	}

	return obj.(*unstructured.Unstructured), err
}

// Delete deletes the resource with the specified name.
func (c *FakeResourceClient) Delete(name string, opts *metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(kTesting.NewDeleteAction(c.Resource, c.Namespace, name), &unstructured.Unstructured{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeResourceClient) DeleteCollection(deleteOptions *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	_, err := c.Fake.
		Invokes(kTesting.NewDeleteCollectionAction(c.Resource, c.Namespace, listOptions), &unstructured.Unstructured{})

	return err
}

// Create creates the provided resource.
func (c *FakeResourceClient) Create(inObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	obj, err := c.Fake.
		Invokes(kTesting.NewCreateAction(c.Resource, c.Namespace, inObj), &unstructured.Unstructured{})

	if obj == nil {
		return nil, err
	}
	return obj.(*unstructured.Unstructured), err
}

// Update updates the provided resource.
func (c *FakeResourceClient) Update(inObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	obj, err := c.Fake.
		Invokes(kTesting.NewUpdateAction(c.Resource, c.Namespace, inObj), &unstructured.Unstructured{})

	if obj == nil {
		return nil, err
	}
	return obj.(*unstructured.Unstructured), err
}

// Watch returns a watch.Interface that watches the resource.
func (c *FakeResourceClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(kTesting.NewWatchAction(c.Resource, c.Namespace, opts))
}

// Patch patches the provided resource.
func (c *FakeResourceClient) Patch(name string, pt types.PatchType, data []byte) (*unstructured.Unstructured, error) {
	obj, err := c.Fake.
		Invokes(kTesting.NewPatchAction(c.Resource, c.Namespace, name, data), &unstructured.Unstructured{})

	if obj == nil {
		return nil, err
	}
	return obj.(*unstructured.Unstructured), err
}

// FakeClientPool provides a fake implementation of dynamic.ClientPool.
// It assumes resource GroupVersions are the same as their corresponding kind GroupVersions.
type FakeClientPool struct {
	kTesting.Fake
}

// ClientForGroupVersionKind returns a client configured for the specified groupVersionResource.
// Resource may be empty.
func (p *FakeClientPool) ClientForGroupVersionResource(resource schema.GroupVersionResource) (dynamic.Interface, error) {
	return p.ClientForGroupVersionKind(resource.GroupVersion().WithKind(""))
}

func NewFakeClientPool(objects ...runtime.Object) *FakeClientPool {
	fakeClientset := fake.NewSimpleClientset(objects...)
	return &FakeClientPool{
		fakeClientset.Fake,
	}
}

// ClientForGroupVersionKind returns a client configured for the specified groupVersionKind.
// Kind may be empty.
func (p *FakeClientPool) ClientForGroupVersionKind(kind schema.GroupVersionKind) (dynamic.Interface, error) {
	// we can just create a new client every time for testing purposes
	return &FakeClient{
		GroupVersion: kind.GroupVersion(),
		Fake:         &p.Fake,
	}, nil
}

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
		Resource: &v1alpha1.ResourceObject{
			Namespace: corev1.NamespaceDefault,
			GroupVersionKind: v1alpha1.GroupVersionKind{
				Version: "v1",
				Kind:    "Pod",
			},
			Source: v1alpha1.ArtifactLocation{
				Inline: &testPod,
			},
			Labels: map[string]string{"test-label": "test-value"},
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
		convey.So(err, convey.ShouldNotBeNil)
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
		fakeclient := soc.clientPool.(*FakeClientPool).Fake
		dynamicClient := dynamicfake.FakeResourceClient{Resource: schema.GroupVersionResource{Version: "v1", Resource: "pods"}, Fake: &fakeclient}

		convey.Convey("Given a pod spec, get a pod object", func() {
			rObj := testTrigger.Template.Resource.DeepCopy()
			rObj.Namespace = "foo"
			pod := &corev1.Pod{
				TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: rObj.Namespace, Name: "my-pod"},
			}
			uObj, err := getUnstructured(pod)
			convey.So(err, convey.ShouldBeNil)

			err = soc.createResourceObject(rObj, testTrigger.ResourceParameters, uObj)
			convey.So(err, convey.ShouldBeNil)

			unstructuredPod, err := dynamicClient.Get(pod.Name, metav1.GetOptions{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(unstructuredPod.GetNamespace(), convey.ShouldEqual, rObj.Namespace)
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
				Src: &v1alpha1.ResourceParameterSource{
					Event: "test-gateway:test",
					Path:  "name",
				},
				Dest: "name",
			},
		}

		testTrigger.ResourceParameters = []v1alpha1.TriggerParameter{
			{
				Src: &v1alpha1.ResourceParameterSource{
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
			rObj := testTrigger.Template.Resource.DeepCopy()
			rObj.Namespace = ""
			pod := &corev1.Pod{
				TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "my-pod-without-namespace"},
			}
			uObj, err := getUnstructured(pod)
			convey.So(err, convey.ShouldBeNil)

			err = soc.createResourceObject(rObj, testTrigger.ResourceParameters, uObj)
			convey.So(err, convey.ShouldBeNil)

			unstructuredPod, err := dynamicClient.Get(pod.Name, metav1.GetOptions{})
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
