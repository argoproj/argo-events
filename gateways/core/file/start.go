package file

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/fsnotify/fsnotify"
	"strings"
	"bytes"
	"encoding/gob"
	"github.com/mitchellh/mapstructure"
	"fmt"
)

// StartConfig runs a configuration
func (ce *FileWatcherConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string
	// mark final gateway state
	defer ce.GatewayConfig.GatewayCleanup(config, &errMessage, err)

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	var fwc *FileWatcherConfig
	err = mapstructure.Decode(config.Data.Config, &fwc)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *fwc).Msg("file configuration")

	go ce.watchFileSystemEvents(fwc, config)

	for {
		select {
		case <-ce.StartChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running.")
			config.Active = true

		case data := <-ce.DataCh:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})

		case <-ce.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.Active = false
			ce.DoneCh <- struct{}{}
			return nil

		case err := <-ce.ErrChan:
			return err
		}
	}
	return nil
}

func (ce *FileWatcherConfigExecutor) watchFileSystemEvents(fwc *FileWatcherConfig, config *gateways.ConfigContext) {
	// create new fs watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ce.ErrChan <- err
		return
	}
	defer watcher.Close()

	// file descriptor to watch must be available in file system. You can't watch an fs descriptor that is not present.
	err = watcher.Add(fwc.Directory)
	if err != nil {
		ce.ErrChan <- err
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("fs watcher has stopped")
				// it means this watcher stopped watching internally
				if config.Active {
					config.Active = false
					ce.ErrChan <- fmt.Errorf("watcher stopped watching file events")
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
					ce.ErrChan <- err
					return
				}
				ce.DataCh <- buff.Bytes()
			}
		case err := <-watcher.Errors:
			ce.ErrChan <- err
		case <-ce.DoneCh:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}