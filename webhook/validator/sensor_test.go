package validator

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	fakeClient "k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestValidateSensor(t *testing.T) {
	dir := "../../examples/sensors"
	files, err := ioutil.ReadDir(dir)
	assert.Nil(t, err)
	for _, file := range files {
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
		assert.Nil(t, err)
		var sensor *v1alpha1.Sensor
		err = yaml.Unmarshal(content, &sensor)
		assert.Nil(t, err)
		client := fakeClient.NewSimpleClientset()
		newSensor := sensor.DeepCopy()
		newSensor.Generation++
		v := NewSensorValidator(client, sensor, newSensor)
		r := v.ValidateCreate(contextWithLogger(t))
		assert.True(t, r.Allowed)
		r = v.ValidateUpdate(contextWithLogger(t))
		assert.True(t, r.Allowed)
	}
}
