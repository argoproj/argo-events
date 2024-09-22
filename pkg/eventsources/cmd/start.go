package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	argoevents "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources"
	"github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

func Start() {
	logger := logging.NewArgoEventsLogger().Named("eventsource")
	encodedEventSourceSpec, defined := os.LookupEnv(v1alpha1.EnvVarEventSourceObject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", v1alpha1.EnvVarEventSourceObject)
	}
	eventSourceSpec, err := base64.StdEncoding.DecodeString(encodedEventSourceSpec)
	if err != nil {
		logger.Fatalw("failed to decode eventsource string", zap.Error(err))
	}
	eventSource := &v1alpha1.EventSource{}
	if err = json.Unmarshal(eventSourceSpec, eventSource); err != nil {
		logger.Fatalw("failed to unmarshal eventsource object", zap.Error(err))
	}

	busConfig := &v1alpha1.BusConfig{}
	encodedBusConfigSpec := os.Getenv(v1alpha1.EnvVarEventBusConfig)
	if len(encodedBusConfigSpec) > 0 {
		busConfigSpec, err := base64.StdEncoding.DecodeString(encodedBusConfigSpec)
		if err != nil {
			logger.Fatalw("failed to decode bus config string", zap.Error(err))
		}
		if err = json.Unmarshal(busConfigSpec, busConfig); err != nil {
			logger.Fatalw("failed to unmarshal bus config object", zap.Error(err))
		}
	}

	ebSubject, defined := os.LookupEnv(v1alpha1.EnvVarEventBusSubject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", v1alpha1.EnvVarEventBusSubject)
	}

	hostname, defined := os.LookupEnv("POD_NAME")
	if !defined {
		logger.Fatal("required environment variable 'POD_NAME' not defined")
	}

	logger = logger.With(logging.LabelEventSourceName, eventSource.Name)
	ctx := logging.WithLogger(signals.SetupSignalHandler(), logger)
	m := metrics.NewMetrics(eventSource.Namespace)
	go m.Run(ctx, fmt.Sprintf(":%d", v1alpha1.EventSourceMetricsPort))

	logger.Infow("starting eventsource server", "version", argoevents.GetVersion())
	adaptor := eventsources.NewEventSourceAdaptor(eventSource, busConfig, ebSubject, hostname, m)

	if err := adaptor.Start(ctx); err != nil {
		logger.Fatalw("failed to start eventsource server", zap.Error(err))
	}
}
