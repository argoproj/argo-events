package file

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/fsnotify/fsnotify"
	"strings"
)

// StartConfig runs a configuration
func (ce *FileWatcherConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	f, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
		return
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *f).Msg("file configuration")

	go ce.watchFileSystemEvents(f, config)

	for {
		select {
		case <-config.StartChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running.")
			config.Active = true

		case data := <-config.DataChan:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *FileWatcherConfigExecutor) watchFileSystemEvents(fwc *FileWatcherConfig, config *gateways.ConfigContext) {
	// create new fs watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		config.ErrChan <- err
		return
	}
	defer watcher.Close()

	// file descriptor to watch must be available in file system. You can't watch an fs descriptor that is not present.
	err = watcher.Add(fwc.Directory)
	if err != nil {
		config.ErrChan <- err
		return
	}

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("k8 event created marking configuration as running")

	config.StartChan <- struct{}{}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("fs watcher has stopped")
				// watcher stopped watching file events
				if config.Active {
					config.Active = false
					config.ErrChan <- fmt.Errorf("watcher stopped watching file events")
					return
				}
			}
			// fwc.Path == event.Name is required because we don't want to send event when .swp files are created
			if fwc.Path == strings.TrimPrefix(event.Name, fwc.Directory) && fwc.Type == event.Op.String() {
				ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Str("event-type", event.Op.String()).Str("descriptor-name", event.Name).Msg("fs event")
				var buff bytes.Buffer
				enc := gob.NewEncoder(&buff)
				err := enc.Encode(event)
				if err != nil {
					config.ErrChan <- err
					return
				}
				config.DataChan <- buff.Bytes()
			}
		case err := <-watcher.Errors:
			config.ErrChan <- err
		case <-config.DoneChan:
			config.ShutdownChan <- struct{}{}
			return
		}
	}
}
