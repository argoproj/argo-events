package webhook

import (
	"github.com/argoproj/argo-events/job"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"go.uber.org/zap"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) (job.Signal, error) {
	abstract.Log.Info("creating signal", zap.String("endpoint", abstract.Webhook.Endpoint))
	return &webhook{
		AbstractSignal: abstract,
	}, nil
}

// Webhook will be added to the executor session
func Webhook(es *job.ExecutorSession) {
	es.AddCoreFactory(v1alpha1.SignalTypeWebhook, &factory{})
}
