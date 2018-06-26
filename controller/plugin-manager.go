package controller

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/argoproj/argo-events/shared"
	"github.com/hashicorp/go-plugin"
)

// PluginManager helps manage the various plugins organized under one directory
type PluginManager struct {
	dir     string
	clients map[string]*plugin.Client
}

// NewPluginManager creates a new PluginManager
func NewPluginManager() (*PluginManager, error) {
	dir := os.Getenv("STREAM_PLUGIN_DIR")
	mgr := PluginManager{
		dir:     dir,
		clients: make(map[string]*plugin.Client),
	}
	plugins, err := plugin.Discover("", dir)
	if err != nil {
		return nil, err
	}
	for _, pluginFile := range plugins {
		c := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: shared.Handshake,
			Plugins:         shared.PluginMap,
			Cmd:             exec.Command(pluginFile),
			AllowedProtocols: []plugin.Protocol{
				plugin.ProtocolNetRPC, plugin.ProtocolGRPC,
			},
		})
		mgr.clients[pluginFile] = c
	}
	return &mgr, nil
}

// Dispense the interface with the given name
// NOTE: assumes the name matches the file name and the plugin name
func (pm *PluginManager) Dispense(name string) (interface{}, error) {
	client, ok := pm.clients[name]
	if !ok {
		return nil, fmt.Errorf("unknown plugin '%s'", name)
	}
	protocol, err := client.Client()
	if err != nil {
		return nil, err
	}
	iface, err := protocol.Dispense(name)
	if err != nil {
		return nil, err
	}
	return iface, nil
}

// Monitor the plugins
func (pm *PluginManager) Monitor(done <-chan struct{}) {
	timer := time.NewTimer(pluginHealthCheckPeriod)
	for {
		select {
		case <-timer.C:
			for _, client := range pm.clients {
				proto, err := client.Client()
				if err != nil {
					panic("failed to retrieve the plugin client protocol")
				}
				err = proto.Ping()
				if err != nil {
					panic("signal plugin client connection failed")
				}
			}
		case <-done:
			timer.Stop()
			return
		}
	}

}

// Close kills all the manager's clients
func (pm *PluginManager) Close() {
	for _, client := range pm.clients {
		client.Kill()
	}
}
