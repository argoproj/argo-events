package sensor

import (
	"testing"
	"k8s.io/client-go/kubernetes/fake"
	kTesting "k8s.io/client-go/testing"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/flowcontrol"
	"github.com/rs/zerolog"
	"os"
	"sync"
	sensorFake "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	discoveryFake "k8s.io/client-go/discovery/fake"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"time"
	"encoding/json"
)

var sensorStr = `apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: webhook-sensor
  labels:
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  repeat: true
  signals:
    - name: test-gateway/test-config
  triggers:
    - name: hello-world-workflow-trigger
      resource:
        group: argoproj.io
        version: v1alpha1
        kind: Workflow
        source:
          inline: |
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                generateName: hello-world-
              spec:
                entrypoint: whalesay
                templates:
                  -
                    container:
                      args:
                        - "hello world"
                      command:
                        - cowsay
                      image: "docker/whalesay:latest"
                    name: whalesay
`

func getSensor() (*v1alpha1.Sensor, error) {
	var sensor v1alpha1.Sensor
	err := yaml.Unmarshal([]byte(sensorStr), &sensor)
	return &sensor, err
}

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

func getsensorExecutionCtx(sensor *v1alpha1.Sensor) *sensorExecutionCtx {
	kubeClientset := fake.NewSimpleClientset()
	return &sensorExecutionCtx{
		kubeClient: kubeClientset,
		discoveryClient: kubeClientset.Discovery().(*discoveryFake.FakeDiscovery),
		clientPool: NewFakeClientPool(),
		log: zerolog.New(os.Stdout).With().Str("sensor-name", sensor.ObjectMeta.Name).Logger(),
		wg: &sync.WaitGroup{},
		sensorClient: sensorFake.NewSimpleClientset(),
		sensor: sensor,
	}
}

func TestSensorExecutionCtx_WatchSignalNotifications(t *testing.T) {
	sensor, err := getSensor()
	assert.Nil(t, err)
	assert.NotNil(t, sensor)
	se := getsensorExecutionCtx(sensor)
	assert.NotNil(t, se)

	// create the sensor
	se.sensor, err = se.sensorClient.ArgoprojV1alpha1().Sensors(sensor.Namespace).Create(sensor)
	assert.Nil(t, err)
	assert.NotNil(t, se.sensor)

	event := &v1alpha1.Event{
		Context: v1alpha1.EventContext{
			CloudEventsVersion: common.CloudEventsVersion,
			EventID:            fmt.Sprintf("%x", "123"),
			ContentType:        "application/json",
			EventTime:          metav1.Time{Time: time.Now().UTC()},
			EventType:          "test",
			EventTypeVersion:   common.CloudEventsVersion,
			Source: &v1alpha1.URI{
				Host: "test-gateway" + "/" + "test-config",
			},
		},
		Payload: []byte(`{
			"x": "abc"
		}`),
	}

	payload, err := json.Marshal(event)
	assert.Nil(t, err)
	assert.NotNil(t, payload)

	valid, err := se.validateSignal(event)
	assert.Nil(t, err)
	assert.Equal(t, true, valid)

	// test persist updates
	se.sensor.Status.Phase = v1alpha1.NodePhaseError
	err = se.persistUpdates()
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.NodePhaseError, se.sensor.Status.Phase)

	// test getK8Event
	k8Event := se.getK8Event("for testing", v1alpha1.NodePhaseComplete, "test-node")
	assert.NotNil(t, k8Event)
	assert.IsType(t, corev1.Event{}, *k8Event)
	assert.NotNil(t, "test-node", k8Event.Name)
	assert.NotNil(t, k8Event.EventTime)

	// test createK8Event
	k8Event, err = se.createK8Event("", "test-gateway/test-config", "", nil)
	assert.Nil(t, err)
	k8Event, err = se.kubeClient.CoreV1().Events(se.sensor.Namespace).Get(k8Event.Name, metav1.GetOptions{})
	assert.NotNil(t, k8Event)
	assert.Equal(t, string(v1alpha1.NodePhaseComplete), k8Event.Action)
	assert.Equal(t, "node completed", k8Event.Reason)
	assert.Equal(t, k8Event.ObjectMeta.Labels[common.LabelEventForSensorNode], "")

	eventWrapper := &v1alpha1.EventWrapper{
		Seen: true,
		Event: *event,
	}
	eventWrapperBytes, err := yaml.Marshal(eventWrapper)
	assert.Nil(t, err)
	assert.NotNil(t, eventWrapperBytes)

	err = se.kubeClient.CoreV1().Events(se.sensor.Namespace).Delete(k8Event.Name, &metav1.DeleteOptions{})
	assert.Nil(t, err)

	k8Event, err = se.createK8Event(string(eventWrapperBytes), "test-gateway/test-config", "", nil)
	assert.Nil(t, err)
	k8Event, err = se.kubeClient.CoreV1().Events(se.sensor.Namespace).Get(k8Event.Name, metav1.GetOptions{})
	assert.NotNil(t, k8Event)
	assert.NotEqual(t, k8Event.ObjectMeta.Labels[common.LabelEventForSensorNode], "")

	k8EventWrapper := k8Event.ObjectMeta.Labels[common.LabelEventForSensorNode]
	var newEventWrapper v1alpha1.EventWrapper
	err = yaml.Unmarshal([]byte(k8EventWrapper), &newEventWrapper)
	assert.Nil(t, err)
	assert.Equal(t, event.Context.Source, newEventWrapper.Context.Source)
	assert.Equal(t, event.Payload, newEventWrapper.Payload)
	assert.NotEqual(t, event.Context.EventTime, newEventWrapper.Context.EventTime)

	err = se.kubeClient.CoreV1().Events(se.sensor.Namespace).Delete(k8Event.Name, &metav1.DeleteOptions{})
	assert.Nil(t, err)

	k8Event, err = se.createK8Event("", "test-gateway/test-config", "test error message", fmt.Errorf("test error"))
	assert.Nil(t, err)
	k8Event, err = se.kubeClient.CoreV1().Events(se.sensor.Namespace).Get(k8Event.Name, metav1.GetOptions{})
	assert.NotNil(t, k8Event)
	assert.Equal(t, string(v1alpha1.NodePhaseError), string(k8Event.Action))

	testNode := v1alpha1.NodeStatus{
		Phase: v1alpha1.NodePhaseNew,
		Name: "test-gateway/test-config",
		StartedAt: metav1.MicroTime{
			Time: time.Now(),
		},
		Type: v1alpha1.NodeTypeSignal,
		DisplayName: "test-gateway/test-config",
	}
	se.sensor.Status.Nodes = map[string]v1alpha1.NodeStatus{
		sensor.NodeID("test-gateway/test-config"): testNode,
	}

	// test markNodePhase
	nodeStatus := se.markNodePhase("test-gateway/test-config", v1alpha1.NodePhaseError, "error phase")
	assert.Nil(t, err)
	assert.NotNil(t, nodeStatus)
	assert.Equal(t, string(v1alpha1.NodePhaseError), string(nodeStatus.Phase))

	// persist updates
	err = se.persistUpdates()
	assert.Nil(t, err)

	delete(se.sensor.Status.Nodes, "")

	// no need to separately test processSignal

	// test updateSensorResource
	err = se.updateSensorResource(k8Event)
	assert.Nil(t, err)
	assert.Equal(t, string(v1alpha1.NodePhaseError), string(se.sensor.Status.Phase))
	k8EventList, err := se.kubeClient.CoreV1().Events(se.sensor.Namespace).List(metav1.ListOptions{})
	for _, updatedK8Event := range k8EventList.Items {
		if updatedK8Event.ObjectMeta.Labels[common.LabelSignalName] == "test-gateway/test-config" {
			assert.Equal(t, "true", updatedK8Event.ObjectMeta.Labels[common.LabelEventSeen])
		}
	}

	err = se.kubeClient.CoreV1().Events(se.sensor.Namespace).Delete(k8Event.Name, &metav1.DeleteOptions{})
	assert.Nil(t, err)

	// test markNodePhase
	nodeStatus = se.markNodePhase( "test-gateway/test-config", v1alpha1.NodePhaseComplete, "completed")
	assert.Nil(t, err)
	assert.NotNil(t, nodeStatus)
	assert.Equal(t, string(v1alpha1.NodePhaseComplete), string(nodeStatus.Phase))

}
