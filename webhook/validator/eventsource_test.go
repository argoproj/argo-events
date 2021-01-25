package validator

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

func TestValidateEventSource(t *testing.T) {
	dir := "../../examples/event-sources"
	files, err := ioutil.ReadDir(dir)
	assert.Nil(t, err)
	eb := fakeEventBus()
	eb.Status.MarkConfigured()
	eb.Status.MarkDeployed("Succeeded", "test")
	_, err = fakeEventBusClient.ArgoprojV1alpha1().EventBus(testNamespace).Create(context.Background(), eb, metav1.CreateOptions{})
	assert.Nil(t, err)
	for _, file := range files {
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
		assert.Nil(t, err)
		var es *v1alpha1.EventSource
		err = yaml.Unmarshal(content, &es)
		assert.Nil(t, err)
		es.Namespace = testNamespace
		newEs := es.DeepCopy()
		newEs.Generation++
		v := NewEventSourceValidator(fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, es, newEs)
		r := v.ValidateCreate(contextWithLogger(t))
		assert.True(t, r.Allowed)
		r = v.ValidateUpdate(contextWithLogger(t))
		assert.True(t, r.Allowed)
	}
	err = fakeEventBusClient.ArgoprojV1alpha1().EventBus(testNamespace).Delete(context.Background(), eb.Name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}
