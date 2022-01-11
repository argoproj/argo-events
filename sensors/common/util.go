package common

import (
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func ApplyEventLabels(labels map[string]string, events map[string]*v1alpha1.Event) error {
	count := 0
	for _, val := range events {
		decodedID, err := hex.DecodeString(val.Context.ID)
		if err != nil {
			return errors.Wrap(err, "failed to decode event ID")
		}

		labelName := fmt.Sprintf("events.argoproj.io/event-%d", count)
		labels[labelName] = string(decodedID)

		count++
	}

	return nil
}

func ApplySensorLabels(labels map[string]string, sensor *v1alpha1.Sensor) {
	sensorGVK := sensor.GroupVersionKind()
	labels["events.argoproj.io/sensor-group"] = sensorGVK.Group
	labels["events.argoproj.io/sensor-version"] = sensorGVK.Version
	labels["events.argoproj.io/sensor-kind"] = sensorGVK.Kind
	labels["events.argoproj.io/sensor-namespace"] = sensor.Namespace
	labels["events.argoproj.io/sensor"] = sensor.Name

	if sensor.Labels != nil && sensor.Labels["app.kubernetes.io/instance"] != "" {
		labels["events.argoproj.io/sensor-app-name"] = sensor.Labels["app.kubernetes.io/instance"]
	}
}
