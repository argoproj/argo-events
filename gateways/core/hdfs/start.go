package hdfs

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/common/fsevent"
	"github.com/argoproj/argo-events/gateways/common/naivewatcher"
	"github.com/colinmarc/hdfs"
)

// EventListener implements Eventing for HDFS events
type EventListener struct {
	// Logger logs stuff
	Logger *logrus.Logger
	// k8sClient is kubernetes client
	K8sClient kubernetes.Interface
}

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
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Info("start processing the event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents listens to HDFS events
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")

	var hdfsEventSource *v1alpha1.HDFSEventSource
	if err := yaml.Unmarshal(eventSource.Value, &hdfsEventSource); err != nil {
		errorCh <- err
		return
	}

	logger.Infoln("setting up HDFS configuration...")
	hdfsConfig, err := createHDFSConfig(listener.K8sClient, hdfsEventSource.Namespace, hdfsEventSource)
	if err != nil {
		errorCh <- err
		return
	}

	logger.Infoln("setting up HDFS client...")
	hdfscli, err := createHDFSClient(hdfsConfig.Addresses, hdfsConfig.HDFSUser, hdfsConfig.KrbOptions)
	if err != nil {
		errorCh <- err
		return
	}
	defer hdfscli.Close()

	logger.Infoln("setting up a new watcher...")
	watcher, err := naivewatcher.NewWatcher(&WatchableHDFS{hdfscli: hdfscli})
	if err != nil {
		errorCh <- err
		return
	}
	defer watcher.Close()

	intervalDuration := 1 * time.Minute
	if hdfsEventSource.CheckInterval != "" {
		d, err := time.ParseDuration(hdfsEventSource.CheckInterval)
		if err != nil {
			errorCh <- err
			return
		}
		intervalDuration = d
	}

	logger.Infoln("started HDFS watcher")
	err = watcher.Start(intervalDuration)
	if err != nil {
		errorCh <- err
		return
	}

	// directory to watch must be available in HDFS. You can't watch a directory that is not present.
	logger.Infoln("adding configured directory to watcher...")
	err = watcher.Add(hdfsEventSource.Directory)
	if err != nil {
		errorCh <- err
		return
	}

	op := fsevent.NewOp(hdfsEventSource.Type)
	var pathRegexp *regexp.Regexp
	if hdfsEventSource.PathRegexp != "" {
		pathRegexp, err = regexp.Compile(hdfsEventSource.PathRegexp)
		if err != nil {
			errorCh <- err
			return
		}
	}

	logger.Infoln("listening to HDFS notifications...")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				logger.Info("HDFS watcher has stopped")
				// watcher stopped watching file events
				errorCh <- fmt.Errorf("HDFS watcher stopped")
				return
			}
			matched := false
			relPath := strings.TrimPrefix(event.Name, hdfsEventSource.Directory)

			if hdfsEventSource.Path != "" && hdfsEventSource.Path == relPath {
				matched = true
			} else if pathRegexp != nil && pathRegexp.MatchString(relPath) {
				matched = true
			}

			if matched && (op&event.Op != 0) {
				logger := logger.WithFields(
					map[string]interface{}{
						"event-type":      event.Op.String(),
						"descriptor-name": event.Name,
					},
				)
				logger.Infoln("received an event")

				logger.Infoln("parsing the event...")
				payload, err := json.Marshal(event)
				if err != nil {
					errorCh <- err
					return
				}

				logger.Infoln("dispatching event on data channel")
				dataCh <- payload
			}
		case err := <-watcher.Errors:
			errorCh <- err
			return
		case <-doneCh:
			return
		}
	}
}
