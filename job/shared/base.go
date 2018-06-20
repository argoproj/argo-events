package shared

import (
	"github.com/argoproj/argo-events/job"
	plugin "github.com/hashicorp/go-plugin"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SIGNAL_PLUGIN",
	MagicCookieValue: "signal",
}

// PluginOpts are options to start the Hashicorp plugin framework
type PluginOpts struct {
	SignalPlugin job.SignalPlugin
}

// GetPluginMap returns the plugin map defined Hashicorp go-plugin.
// The reserved parameter should only be used by the RPC receiver (the plugin).
// Otherwise, reserved should be nil for the RPC sender (the executor).
func GetPluginMap(reserved *PluginOpts) map[string]plugin.Plugin {
	var signalerObj job.SignalPlugin

	if reserved != nil {
		signalerObj = reserved.SignalPlugin
	}

	return map[string]plugin.Plugin{
		"NATS": &signalerObj,
	}
}
