package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	v1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

func main() {
	encodedEventSourceSpec, defined := os.LookupEnv(common.EnvVarEventSourceObject)
	if !defined {
		panic(errors.Errorf("required environment variable '%s' not defined", common.EnvVarEventSourceObject))
	}
	eventSourceSpec, err := base64.StdEncoding.DecodeString(encodedEventSourceSpec)
	if err != nil {
		panic(errors.Errorf("failed to decode eventsource string. err: %v", err))
	}
	eventSource := &v1alpha1.EventSource{}
	if err = json.Unmarshal(eventSourceSpec, eventSource); err != nil {
		panic(errors.Errorf("failed to unmarshal eventsource object. err: %v", err))
	}

	busConfig := &eventbusv1alpha1.BusConfig{}
	encodedBusConfigSpec := os.Getenv(common.EnvVarEventBusConfig)
	if len(encodedBusConfigSpec) > 0 {
		busConfigSpec, err := base64.StdEncoding.DecodeString(encodedBusConfigSpec)
		if err != nil {
			panic(errors.Errorf("failed to decode bus config string. err: %+v", err))
		}
		if err = json.Unmarshal(busConfigSpec, busConfig); err != nil {
			panic(errors.Errorf("failed to unmarshal bus config object. err: %+v", err))
		}
	}

	ebSubject, defined := os.LookupEnv(common.EnvVarEventBusSubject)
	if !defined {
		panic(errors.Errorf("required environment variable '%s' not defined", common.EnvVarEventBusSubject))
	}

	hostname, defined := os.LookupEnv("POD_NAME")
	if !defined {
		panic(errors.New("required environment variable 'POD_NAME' not defined"))
	}

	adaptor := eventsources.NewEventSourceAdaptor(eventSource, busConfig, ebSubject, hostname)
	logger := logging.NewArgoEventsLogger()
	ctx := logging.WithLogger(context.Background(), logger)
	stopCh := signals.SetupSignalHandler()
	if err := adaptor.Start(ctx, stopCh); err != nil {
		logger.WithError(err).Errorln("failed to start eventsource server")
		os.Exit(-1)
	}
}
