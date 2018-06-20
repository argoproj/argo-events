package job

import (
	"net/rpc"

	v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

type RPCClient struct{ client *rpc.Client }

func (c *RPCClient) Start(signal *v1alpha1.Signal) (<-chan Event, error) {
	var resp <-chan Event
	err := c.client.Call("Plugin.Start", map[string]interface{}{
		"signal": signal,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RPCClient) Stop() error {
	var resp error
	err := c.client.Call("Plugin.Stop", map[string]interface{}{}, &resp)
	if err != nil {
		return err
	}
	return resp
}

// RPCServer is the RPC server that RPCClient talks to, conforming to
// the requirements of net/rpc
type RPCServer struct {
	// This is the real implementation
	Impl Signaler
}

func (s *RPCServer) Start(args map[string]interface{}, resp *interface{}) error {
	events, err := s.Impl.Start(args["signal"].(*v1alpha1.Signal))
	*resp = events
	return err
}

func (s *RPCServer) Stop(args map[string]interface{}, resp *interface{}) error {
	return s.Impl.Stop()
}
