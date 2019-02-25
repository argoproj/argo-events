package hdfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/common/naivewatcher"
	"github.com/colinmarc/hdfs"
)

// WatchableHDFS wraps hdfs.Client for naivewatcher
type WatchableHDFS struct {
	hdfscli *hdfs.Client
}

// Walk walks a directory
func (w *WatchableHDFS) Walk(root string, walkFn filepath.WalkFunc) error {
	return w.hdfscli.Walk(root, walkFn)
}

// GetFileID returns the file ID
func (w *WatchableHDFS) GetFileID(fi os.FileInfo) interface{} {
	return fi.Name()
	// FIXME: Use HDFS File ID once it's exposed
	//   https://github.com/colinmarc/hdfs/pull/171
	// return fi.Sys().(*hadoop_hdfs.HdfsFileStatusProto).GetFileID()
}

// StartEventSource starts an event source
func (ese *EventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("activating event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}
	gwc := config.(*GatewayConfig)

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(gwc, eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
}

func (ese *EventSourceExecutor) listenEvents(config *GatewayConfig, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	hdfsConfig, err := createHDFSConfig(ese.Clientset, ese.Namespace, &config.GatewayClientConfig)
	if err != nil {
		errorCh <- err
		return
	}

	hdfscli, err := createHDFSClient(hdfsConfig.Addresses, hdfsConfig.HDFSUser, hdfsConfig.KrbOptions)
	if err != nil {
		errorCh <- err
		return
	}
	defer hdfscli.Close()

	// create new watcher
	watcher, err := naivewatcher.NewWatcher(&WatchableHDFS{hdfscli: hdfscli})
	if err != nil {
		errorCh <- err
		return
	}
	defer watcher.Close()

	intervalDuration := 1 * time.Minute
	if config.CheckInterval != "" {
		d, err := time.ParseDuration(config.CheckInterval)
		if err != nil {
			errorCh <- err
			return
		}
		intervalDuration = d
	}

	err = watcher.Start(intervalDuration)
	if err != nil {
		errorCh <- err
		return
	}

	// directory to watch must be available in HDFS. You can't watch a directory that is not present.
	err = watcher.Add(config.Directory)
	if err != nil {
		errorCh <- err
		return
	}

	op := naivewatcher.NewOp(config.Type)
	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("starting to watch to HDFS notifications")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				ese.Log.Info().Str("event-source", eventSource.Name).Msg("HDFS watcher has stopped")
				// watcher stopped watching file events
				errorCh <- fmt.Errorf("HDFS watcher stopped")
				return
			}
			if config.Path == strings.TrimPrefix(event.Name, config.Directory) && (op&event.Op != 0) {
				ese.Log.Debug().Str("config-key", eventSource.Name).Str("event-type", event.Op.String()).Str("descriptor-name", event.Name).Msg("HDFS event")
				dataCh <- []byte(fmt.Sprintf("%v", event))
			}
		case err := <-watcher.Errors:
			errorCh <- err
			return
		case <-doneCh:
			return
		}
	}
}
