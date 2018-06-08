package webhook

import (
	"github.com/blackrock/axis/job"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	"go.uber.org/zap"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) job.Signal {
	abstract.Log.Info("creating signal", zap.String("endpoint", abstract.Webhook.Endpoint))
	return &webhook{
		AbstractSignal: abstract,
	}
}

// Webhook will be added to the executor session
func Webhook(es *job.ExecutorSession) {
	es.AddFactory(v1alpha1.SignalTypeWebhook, &factory{})
}
