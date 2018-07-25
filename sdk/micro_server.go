package sdk

import (
	"context"
	"errors"
	io "io"
	"log"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	google_protobuf "github.com/golang/protobuf/ptypes/empty"
)

type microSignalServer struct {
	impl Listener
}

var ack = &EventContext{Done: true}

// NewMicroSignalServer creates a Micro compatible SignalServer from the Listener implementation
func NewMicroSignalServer(lis Listener) SignalServer {
	return &microSignalServer{impl: lis}
}

// Ping implements the SignalServiceHandler interface
func (m *microSignalServer) Ping(ctx context.Context, in *google_protobuf.Empty, out *google_protobuf.Empty) error {
	out = in
	return ctx.Err()
}

// Listen implements the SignalServiceHandler interface
func (m *microSignalServer) Listen(ctx context.Context, stream SignalService_ListenStream) error {
	done := make(chan struct{})

	// perform the initial handshake
	events, err := m.handshake(stream, done)
	if err != nil {
		return err
	}

	// start the receive goroutine to monitor context updates
	go func() {
		for {
			sigCtx, err := stream.Recv()
			// check if the signal is same and was updated/different?
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Panicf("encountered error on stream recv: %s", err)
			}
			if sigCtx.Done {
				close(done)
			}
		}
	}()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				// finished
				return stream.Close()
			}
			err := stream.Send(&EventContext{Event: event})
			if err != nil {
				log.Printf("microSignalServer failed to send event: %s", err)
				return err
			}
		case <-ctx.Done():
			// TODO: think about swallowing context errors and just closing the done channel, but keep the program from panicing
			close(done)
			return ctx.Err()
		}
	}
}

// Handshake performs the initial handshake for the signal server.
// This blocks until the initial receive and send have completed or an error is encountered.
func (m *microSignalServer) handshake(stream SignalService_ListenStream, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	sigCtx, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	if sigCtx.Done {
		return nil, errors.New("signal context done before started")
	}
	events, err := m.impl.Listen(sigCtx.Signal.DeepCopy(), done)
	if err != nil {
		return nil, err
	}
	err = stream.Send(ack)
	if err != nil {
		return nil, err
	}
	return events, nil
}
