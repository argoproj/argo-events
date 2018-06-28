/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shared

import (
	"net/rpc"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	plugin "github.com/hashicorp/go-plugin"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

const (
	CloudEventsVersion       = "v1.0"
	ContextExtensionErrorKey = "error"
	SignalPluginName         = "signaler"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SIGNAL_PLUGIN",
	MagicCookieValue: "signal",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	SignalPluginName: &signalPlugin{},
}

// Signaler is the interface for signaling
type Signaler interface {
	Start(*v1alpha1.Signal) (<-chan *v1alpha1.Event, error)
	Stop() error
}

// ArtifactSignaler is the interface for signaling with artifacts
// In addition to including the basic Signaler interface, this also
// enables access to read an artifact object to include in the event data payload
type ArtifactSignaler interface {
	Signaler
	// todo: change to use io.Reader and io.Closer interfaces?
	Read(*v1alpha1.ArtifactLocation, string) ([]byte, error)
}

// NewPlugin creates a base signal plugin
func NewPlugin(impl Signaler) plugin.Plugin {
	return &signalPlugin{Impl: impl}
}

// signalPlugin is the implementation of plugin.Plugin so we can serve/consume this.
type signalPlugin struct {
	Impl Signaler
}

func (p *signalPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &RPCServer{Impl: p.Impl}, nil
}

func (p *signalPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCClient{client: c}, nil
}

func (p *signalPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterSignalServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *signalPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: NewSignalClient(c)}, nil
}
