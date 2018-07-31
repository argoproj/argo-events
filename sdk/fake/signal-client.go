package fake

import (
	"context"
	"io"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	"github.com/micro/go-micro/client"
)

// SignalClient implements the sdk.SignalClient
// also contains a Send() method which should be invoked via a separate go-routine
// if your test is single-threaded to prevent blocking channels
type SignalClient interface {
	sdk.SignalClient
	Generate(*v1alpha1.Event)
}

// implements sdk.SignalService_ListenService as a simple loop
type simpleLoopListenService struct {
	events chan *v1alpha1.Event
}

func (*simpleLoopListenService) SendMsg(interface{}) error {
	return nil
}

func (*simpleLoopListenService) RecvMsg(interface{}) error {
	return nil
}

func (f *simpleLoopListenService) Close() error {
	close(f.events)
	return nil
}

func (*simpleLoopListenService) Send(*sdk.SignalContext) error {
	return nil
}

func (f *simpleLoopListenService) Recv() (*sdk.EventContext, error) {
	event, ok := <-f.events
	if !ok {
		return nil, io.EOF
	}
	return &sdk.EventContext{Event: event}, nil
}

type fakeSignalClient struct {
	loop *simpleLoopListenService
}

// NewClient returns a new fake signal client
func NewClient() SignalClient {
	return &fakeSignalClient{
		loop: &simpleLoopListenService{
			events: make(chan *v1alpha1.Event),
		},
	}
}

func (*fakeSignalClient) Ping(context.Context) error {
	return nil
}

func (f *fakeSignalClient) Listen(context.Context, *v1alpha1.Signal, ...client.CallOption) (sdk.SignalService_ListenService, error) {
	return f.loop, nil
}

func (*fakeSignalClient) Handshake(*v1alpha1.Signal, sdk.SignalService_ListenService) error {
	return nil
}

// Generate allows us to produce fake events for testing purposes
func (f *fakeSignalClient) Generate(e *v1alpha1.Event) {
	go func() { f.loop.events <- e.DeepCopy() }()
}
