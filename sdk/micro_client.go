package sdk

import (
	"errors"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	google_protobuf "github.com/golang/protobuf/ptypes/empty"
	client "github.com/micro/go-micro/client"
	context "golang.org/x/net/context"
)

type microSignalClient struct {
	impl SignalService
}

// Terminate is SignalContext to send in order to stop the signal
var Terminate = &SignalContext{Done: true}

// NewMicroSignalClient creates a Micro compatible SignalClient from a micro.Client
func NewMicroSignalClient(name string, c client.Client) SignalClient {
	return &microSignalClient{impl: NewSignalService(name, c)}
}

// Ping the signal service
func (m *microSignalClient) Ping(ctx context.Context) error {
	_, err := m.impl.Ping(ctx, &google_protobuf.Empty{})
	return err
}

// Listen to the signal
func (m *microSignalClient) Listen(ctx context.Context, signal *v1alpha1.Signal, opts ...client.CallOption) (SignalService_ListenService, error) {
	// #200 - gRPC server defaults to 5s request timeout on stream. setting -1 means indefinite for streams.
	opts = append(opts, client.WithRequestTimeout(-1))
	stream, err := m.impl.Listen(ctx, opts...)
	if err != nil {
		return nil, err
	}
	err = m.handshake(signal, stream)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// Handshake performs the initial signal handshaking with the server
func (m *microSignalClient) handshake(signal *v1alpha1.Signal, stream SignalService_ListenService) error {
	err := stream.Send(&SignalContext{Signal: signal})
	if err != nil {
		return err
	}
	ack, err := stream.Recv()
	if err != nil {
		return err
	}
	if ack.Done {
		return nil
	}
	return errors.New("handshake ack failed")
}
