package log

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

type LogTrigger struct {
	Sensor      *v1alpha1.Sensor
	Trigger     *v1alpha1.Trigger
	Logger      *zap.Logger
	LastLogTime time.Time
}

func NewLogTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.Logger) (*LogTrigger, error) {
	return &LogTrigger{Sensor: sensor, Trigger: trigger, Logger: logger}, nil
}

func (t *LogTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.Log, nil
}

func (t *LogTrigger) ApplyResourceParameters(_ map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	return resource, nil
}

func (t *LogTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	log, ok := resource.(*v1alpha1.LogTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the fetched trigger resource")
	}
	if t.shouldLog(log) {
		for dependencyName, event := range events {
			t.Logger.Info(
				event.DataString(),
				zap.String("triggerName", t.Trigger.Template.Name),
				zap.String("dependencyName", dependencyName),
				zap.Any("eventContext", event.Context),
			)
		}
		t.LastLogTime = time.Now()
	}
	return nil, nil
}

func (t *LogTrigger) shouldLog(log *v1alpha1.LogTrigger) bool {
	return time.Now().After(t.LastLogTime.Add(log.GetInterval()))
}

func (t *LogTrigger) ApplyPolicy(context.Context, interface{}) error {
	return nil
}
