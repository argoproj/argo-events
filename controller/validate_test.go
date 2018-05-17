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

package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
)

func TestValidateSensor(t *testing.T) {
	if signalPassed := t.Run("TestValidateSignals", validateSignal); !signalPassed {
		t.Fail()
	}
	if triggerPassed := t.Run("TestValidateTriggers", validateTrigger); !triggerPassed {
		t.Fail()
	}
	if complete := t.Run("TestValidSpec", validSpec); !complete {
		t.Fail()
	}
}

func validateSignal(t *testing.T) {
	// no signals defined
	noSignalsSensor := &v1alpha1.Sensor{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	err := validateSensor(noSignalsSensor)
	if err == nil {
		t.Errorf("Sensor incorrectly passed validation with no signals")
	} else {
		t.Logf("Sensor correctly failed validation with no signals")
	}

	// unknown signal type
	unknownSignalSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v1alpha1.SensorSpec{
			Signals: []v1alpha1.Signal{
				{
					Name: "signal",
				},
			},
			Triggers: []v1alpha1.Trigger{
				{
					Name: "trigger",
				},
			},
		},
	}
	err = validateSensor(unknownSignalSensor)
	if err == nil {
		t.Errorf("Sensor incorrectly passed validation with unknown signal type")
	} else {
		t.Logf("Sensor correctly failed validation with unknown signal type")
	}
}

func validateTrigger(t *testing.T) {
	// no triggers defined
	noTriggerSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v1alpha1.SensorSpec{
			Signals: []v1alpha1.Signal{
				{
					Name:     "test",
					Artifact: &v1alpha1.ArtifactSignal{},
				},
			},
		},
	}
	err := validateSensor(noTriggerSensor)
	if err == nil {
		t.Errorf("Sensor incorrectly passed validation with no triggers")
		return
	}
	t.Logf("Sensor correctly failed validation with no triggers")

	// unknown triggers defined
	unknownTriggerSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v1alpha1.SensorSpec{
			Signals: []v1alpha1.Signal{
				{
					Name:     "test",
					Artifact: &v1alpha1.ArtifactSignal{},
				},
			},
			Triggers: []v1alpha1.Trigger{
				{
					Name: "test1",
				},
			},
		},
	}
	err = validateSensor(unknownTriggerSensor)
	if err == nil {
		t.Errorf("Sensor incorrectly passed validation with unknown trigger type")
		return
	}
	t.Logf("Sensor correctly failed validation with unknown trigger type")

}

func validSpec(t *testing.T) {
	successSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v1alpha1.SensorSpec{
			Signals: []v1alpha1.Signal{
				{
					Name:     "signal",
					Artifact: &v1alpha1.ArtifactSignal{},
				},
			},
			Triggers: []v1alpha1.Trigger{
				{
					Name:    "trigger",
					Message: &v1alpha1.Message{},
				},
			},
		},
	}
	err := validateSensor(successSensor)
	if err != nil {
		t.Errorf("Sensor incorrectly failed validation")
		return
	}
	t.Logf("Sensor correctly passed validation")
}
