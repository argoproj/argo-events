package log

import (
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

type LogTrigger struct {
	Sensor  *v1alpha1.Sensor
	Trigger *v1alpha1.Trigger
	Logger  *zap.Logger
}

func NewLogTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.Logger) (*LogTrigger, error) {
	return &LogTrigger{Sensor: sensor, Trigger: trigger, Logger: logger}, nil
}

func (t *LogTrigger) FetchResource() (interface{}, error) {
	return t.Trigger.Template.Log, nil
}

func (t *LogTrigger) ApplyResourceParameters(_ map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	return resource, nil
}

func (t *LogTrigger) Execute(events map[string]*v1alpha1.Event, _ interface{}) (interface{}, error) {
	for dependencyName, event := range events {
		t.Logger.Info(
			event.DataString(),
			zap.String("triggerName", t.Trigger.Template.Name),
			zap.String("dependencyName", dependencyName),
			zap.Any("eventContext", event.Context),
		)
	}
	return nil, nil
}

func (t *LogTrigger) ApplyPolicy(interface{}) error {
	return nil
}
