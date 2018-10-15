package file

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
	"strings"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// fileWatcher implements ConfigExecutor interface
type fileWatcher struct{}

// fileWatcherConfig contains configuration information for this gateway
type fileWatcherConfig struct {
	// Directory to watch for events
	Directory string
	// Path is relative path of object to watch with respect to the directory
	Path string
	// Type of file operations to watch
	// Refer https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go for more information
	Type string
}

// StartConfig runs a configuration
func (fw *fileWatcher) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string
	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("started operating on configuration")

	var fwc *fileWatcherConfig
	err = yaml.Unmarshal([]byte(config.Data.Config), &fwc)
	if err != nil {
		errMessage = "failed to parse configuration"
		return err
	}
	// create new fs watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errMessage = "failed to create new file system watcher"
		return err
	}
	defer watcher.Close()

	// file descriptor to watch must be available in file system. You can't watch an fs descriptor that is not present.
	err = watcher.Add(fwc.Directory)
	if err != nil {
		errMessage = fmt.Sprintf("failed to add path %s to fs watcher", fwc.Path)
		return err
	}

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running...")
	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("k8 event created marking configuration as running")

	// start listening fs notifications
NotificationListener:
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("fs watcher has stopped")
				config.Active = false
				break NotificationListener
			}
			// fwc.Path == event.Name is required because we don't want to send event when .swp files are created
			if fwc.Path == strings.TrimPrefix(event.Name, fwc.Directory) && fwc.Type == event.Op.String() {
				gatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Str("event-type", event.Op.String()).Str("descriptor-name", event.Name).Msg("fs event")
				var buff bytes.Buffer
				enc := gob.NewEncoder(&buff)
				err := enc.Encode(event)
				if err != nil {
					gatewayConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to encode fs event")
					errMessage = "failed to encode fs event"
					config.Active = false
					break NotificationListener
				} else {
					gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
					gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
						Src:     config.Data.Src,
						Payload: buff.Bytes(),
					})
				}
			}
		case e, ok := <-watcher.Errors:
			if !ok {
				gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("fs watcher has stopped")
				config.Active = false
				break NotificationListener
			}
			err = e
			errMessage = "error occurred in fs watcher"
			config.Active = false
			break NotificationListener
		case <-config.StopCh:
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")
			config.Active = false
			break NotificationListener
		}
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now stopped.")
	return nil
}

// StopConfig deactivates a configuration
func (fw *fileWatcher) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

func main() {
	err := gatewayConfig.TransformerReadinessProbe()
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to connect to gateway transformer")
	}
	_, err = gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch k8 events for gateway configuration state updates")
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &fileWatcher{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}
