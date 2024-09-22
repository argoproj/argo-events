package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	argoevents "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/metrics"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

func Start() {
	logger := logging.NewArgoEventsLogger().Named("eventsource")
	encodedEventSourceSpec, defined := os.LookupEnv(sharedutil.EnvVarEventSourceObject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", sharedutil.EnvVarEventSourceObject)
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
	encodedBusConfigSpec := os.Getenv(sharedutil.EnvVarEventBusConfig)
	if len(encodedBusConfigSpec) > 0 {
		busConfigSpec, err := base64.StdEncoding.DecodeString(encodedBusConfigSpec)
		if err != nil {
			logger.Fatalw("failed to decode bus config string", zap.Error(err))
		}
		if err = json.Unmarshal(busConfigSpec, busConfig); err != nil {
			logger.Fatalw("failed to unmarshal bus config object", zap.Error(err))
		}
	}

	ebSubject, defined := os.LookupEnv(sharedutil.EnvVarEventBusSubject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", sharedutil.EnvVarEventBusSubject)
	}

	hostname, defined := os.LookupEnv("POD_NAME")
	if !defined {
		logger.Fatal("required environment variable 'POD_NAME' not defined")
	}

	logger = logger.With(logging.LabelEventSourceName, eventSource.Name)
	ctx := logging.WithLogger(signals.SetupSignalHandler(), logger)
	m := metrics.NewMetrics(eventSource.Namespace)
	go m.Run(ctx, fmt.Sprintf(":%d", sharedutil.EventSourceMetricsPort))

	logger.Infow("starting eventsource server", "version", argoevents.GetVersion())
	adaptor := eventsources.NewEventSourceAdaptor(eventSource, busConfig, ebSubject, hostname, m)

	if err := adaptor.Start(ctx); err != nil {
		logger.Fatalw("failed to start eventsource server", zap.Error(err))
	}
}
