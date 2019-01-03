package nats

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/nats-io/go-nats"
)

// Runs a configuration
func (ce *NatsConfigExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ce.GatewayConfig.Log.Info().Str("event-source-name", *eventSource.Name).Msg("operating on event source")
	n, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ce.listenEvents(n, eventSource, dataCh, errorCh, doneCh)

	return gateways.ConsumeEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ce.Log)
}

func (ce *NatsConfigExecutor) listenEvents(n *natsConfig, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	nc, err := nats.Connect(n.URL)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("url", n.URL).Err(err).Msg("connection failed")
		errorCh <- err
		return
	}

	_, err = nc.Subscribe(n.Subject, func(msg *nats.Msg) {
		dataCh <- msg.Data
	})
	if err != nil {
		errorCh <- err
		return
	}
	nc.Flush()
	if err := nc.LastError(); err != nil {
		errorCh <- err
		return
	}

	<-doneCh
}
