package shared

import (
	"context"
	fmt "fmt"
	"io"
	"log"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	empty "github.com/golang/protobuf/ptypes/empty"
)

// GRPCClient is an implementation of Stream that talks over gRPC.
type GRPCClient struct{ client SignalClient }

func (c *GRPCClient) Start(signal *v1alpha1.Signal) (<-chan Event, error) {
	ch := make(chan Event, 32)
	stream, err := c.client.Start(context.Background(), signal)
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				// we are done
				log.Printf("GRPCClient: received io.EOF, finishing sending")
				return
			}
			if err != nil {
				// error during processing - should we use callbacks here?
				panic(err)
			}
			ch <- *event
		}
	}()
	return ch, nil
}

func (c *GRPCClient) Stop() error {
	_, err := c.client.Stop(context.Background(), &empty.Empty{})
	return err
}

// GRPCServer is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl Signaler
}

func (s *GRPCServer) Start(signal *v1alpha1.Signal, stream Signal_StartServer) error {
	events, err := s.Impl.Start(signal)
	if err != nil {
		return err
	}
	for {
		select {
		case event, ok := <-events:
			if !ok {
				// finished
				return nil
			}
			err := stream.Send(&event)
			if err != nil {
				return err
			}
		case <-stream.Context().Done():
			return fmt.Errorf("GRPCServer: context finished")
		}
	}
}

func (s *GRPCServer) Stop(context.Context, *empty.Empty) (*empty.Empty, error) {
	return &empty.Empty{}, s.Impl.Stop()
}
