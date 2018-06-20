package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/argoproj/argo-events/job"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/golang/protobuf/ptypes"
	plugin "github.com/hashicorp/go-plugin"
	natsio "github.com/nats-io/go-nats"
)

// NATS is a plugin for a stream signal
type NATS struct {
	natsConn         *natsio.Conn
	natsSubscription *natsio.Subscription
	msgCh            chan *natsio.Msg
	stop             chan struct{}
}

// Start NATS signal
func (n *NATS) Start(signal *v1alpha1.Signal) (<-chan job.Event, error) {
	// parse out the attributes
	subject, ok := signal.Stream.Attributes["subject"]
	if !ok {
		return nil, job.ErrMissingRequiredAttribute
	}
	var err error
	n.natsConn, err = natsio.Connect(signal.Stream.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats cluster url %s. Cause: %+v", signal.Stream.URL, err.Error())
	}
	n.natsSubscription, err = n.natsConn.ChanSubscribe(subject, n.msgCh)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to nats subject %s. Cause: %+v", subject, err.Error())
	}
	events := make(chan job.Event)
	go n.listen(events)
	return events, nil
}

// Stop NATS signal
func (n *NATS) Stop() error {
	defer n.natsConn.Close()
	defer close(n.msgCh)
	log.Printf("stopping signal")
	n.stop <- struct{}{}
	return n.natsSubscription.Unsubscribe()
}

func main() {
	nats := &NATS{
		msgCh: make(chan *natsio.Msg),
		stop:  make(chan struct{}),
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: job.Handshake,
		Plugins: map[string]plugin.Plugin{
			"NATS": &job.SignalPlugin{Impl: nats},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func (n *NATS) listen(events chan job.Event) {
	id := 0
	for {
		select {
		case natsMsg := <-n.msgCh:
			event := &job.Event{
				Context: &job.EventContext{
					EventType:          "com.github.nats-io.pub",
					EventTypeVersion:   "",
					CloudEventsVersion: "v1.0",
					Source:             &job.URI{},
					EventID:            natsMsg.Subject + "-" + strconv.Itoa(id),
					EventTime:          ptypes.TimestampNow(),
					SchemaURL:          &job.URI{},
					ContentType:        "",
					Extensions:         make(map[string]string),
				},
				Data: natsMsg.Data,
			}
			log.Printf("sending nat event")
			events <- *event
		case <-n.stop:
			close(events)
			return
		}
	}
}
