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

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
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

var testWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
generateName: hello-world-
spec:
entrypoint: whalesay
templates:
- name: whalesay
container:
    args:
    - "hello world"
    command:
    - cowsay
    image: "docker/whalesay:latest"
`

var testTrigger = v1alpha1.Trigger{
	Name: "sample",
	Resource: &v1alpha1.ResourceObject{
		Namespace: corev1.NamespaceDefault,
		GroupVersionKind: v1alpha1.GroupVersionKind{
			Group:   "argoproj.io",
			Version: "v1alpha1",
			Kind:    "workflow",
		},
		Source: v1alpha1.ArtifactLocation{
			Inline: &testWf,
		},
		Labels: map[string]string{"test-label": "test-value"},
	},
}

func TestProcessTrigger(t *testing.T) {
	triggers := make([]v1alpha1.Trigger, 1)
	triggers[0] = testTrigger
	testSensor, err := getSensor()
	assert.Nil(t, err)
	testSensor.Spec.Triggers = triggers
	soc := getsensorExecutionCtx(testSensor)
	err = soc.executeTrigger(testTrigger)
	assert.NotNil(t, err)
}
