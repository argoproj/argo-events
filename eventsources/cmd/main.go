package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	v1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

func main() {
	logger := logging.NewArgoEventsLogger().Named("eventsource")
	encodedEventSourceSpec, defined := os.LookupEnv(common.EnvVarEventSourceObject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", common.EnvVarEventSourceObject)
	}
	eventSourceSpec, err := base64.StdEncoding.DecodeString(encodedEventSourceSpec)
	if err != nil {
		logger.Desugar().Fatal("failed to decode eventsource string", zap.Error(err))
	}
	eventSource := &v1alpha1.EventSource{}
	if err = json.Unmarshal(eventSourceSpec, eventSource); err != nil {
		logger.Desugar().Fatal("failed to unmarshal eventsource object", zap.Error(err))
	}

	busConfig := &eventbusv1alpha1.BusConfig{}
	encodedBusConfigSpec := os.Getenv(common.EnvVarEventBusConfig)
	if len(encodedBusConfigSpec) > 0 {
		busConfigSpec, err := base64.StdEncoding.DecodeString(encodedBusConfigSpec)
		if err != nil {
			logger.Desugar().Fatal("failed to decode bus config string", zap.Error(err))
		}
		if err = json.Unmarshal(busConfigSpec, busConfig); err != nil {
			logger.Desugar().Fatal("failed to unmarshal bus config object", zap.Error(err))
		}
	}

	ebSubject, defined := os.LookupEnv(common.EnvVarEventBusSubject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", common.EnvVarEventBusSubject)
	}

	hostname, defined := os.LookupEnv("POD_NAME")
	if !defined {
		logger.Fatal("required environment variable 'POD_NAME' not defined")
	}

	adaptor := eventsources.NewEventSourceAdaptor(eventSource, busConfig, ebSubject, hostname)
	logger = logger.With(logging.LabelEventSourceName, eventSource.Name)
	ctx := logging.WithLogger(context.Background(), logger)
	stopCh := signals.SetupSignalHandler()
	if err := adaptor.Start(ctx, stopCh); err != nil {
		logger.Desugar().Fatal("failed to start eventsource server", zap.Error(err))
	}
}
