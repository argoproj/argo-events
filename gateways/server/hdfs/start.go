package hdfs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/gateways/server/common/fsevent"
	"github.com/argoproj/argo-events/gateways/server/common/naivewatcher"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/colinmarc/hdfs"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for HDFS events
type EventListener struct {
	// Logger logs stuff
	Logger *logrus.Logger
	// k8sClient is kubernetes client
	K8sClient kubernetes.Interface
	Namespace string
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
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	channels := server.NewChannels()

	go server.HandleEventsFromEventSource(eventSource.Name, eventStream, channels, listener.Logger)

	defer func() {
		channels.Stop <- struct{}{}
	}()

	if err := listener.listenEvents(eventSource, channels); err != nil {
		listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to listen to events")
		return err
	}

	return nil
}

// listenEvents listens to HDFS events
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var hdfsEventSource *v1alpha1.HDFSEventSource
	if err := yaml.Unmarshal(eventSource.Value, &hdfsEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse event source %s", eventSource.Name)
	}

	if hdfsEventSource.Namespace == "" {
		hdfsEventSource.Namespace = listener.Namespace
	}

	logger.Infoln("setting up HDFS configuration...")
	hdfsConfig, err := createHDFSConfig(listener.K8sClient, hdfsEventSource.Namespace, hdfsEventSource)
	if err != nil {
		return errors.Wrapf(err, "failed to create HDFS configuration for %s", eventSource.Name)
	}

	logger.Infoln("setting up HDFS client...")
	hdfscli, err := createHDFSClient(hdfsConfig.Addresses, hdfsConfig.HDFSUser, hdfsConfig.KrbOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to create the HDFS client for %s", eventSource.Name)
	}
	defer hdfscli.Close()

	logger.Infoln("setting up a new watcher...")
	watcher, err := naivewatcher.NewWatcher(&WatchableHDFS{hdfscli: hdfscli})
	if err != nil {
		return errors.Wrapf(err, "failed to create the HDFS watcher for %s", eventSource.Name)
	}
	defer watcher.Close()

	intervalDuration := 1 * time.Minute
	if hdfsEventSource.CheckInterval != "" {
		d, err := time.ParseDuration(hdfsEventSource.CheckInterval)
		if err != nil {
			return errors.Wrapf(err, "failed to parse the check in interval for %s", eventSource.Name)
		}
		intervalDuration = d
	}

	logger.Infoln("started HDFS watcher")
	err = watcher.Start(intervalDuration)
	if err != nil {
		return errors.Wrapf(err, "failed to start the watcher for %s", eventSource.Name)
	}

	// directory to watch must be available in HDFS. You can't watch a directory that is not present.
	logger.Infoln("adding configured directory to watcher...")
	err = watcher.Add(hdfsEventSource.Directory)
	if err != nil {
		return errors.Wrapf(err, "failed to add directory %s for %s", hdfsEventSource.Directory, eventSource.Name)
	}

	op := fsevent.NewOp(hdfsEventSource.Type)
	var pathRegexp *regexp.Regexp
	if hdfsEventSource.PathRegexp != "" {
		pathRegexp, err = regexp.Compile(hdfsEventSource.PathRegexp)
		if err != nil {
			return errors.Wrapf(err, "failed to compile the path regex %s for %s", hdfsEventSource.PathRegexp, eventSource.Name)
		}
	}

	logger.Infoln("listening to HDFS notifications...")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				logger.Info("HDFS watcher has stopped")
				// watcher stopped watching file events
				return errors.Errorf("watcher has been stopped for %s", eventSource.Name)
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
					logger.WithError(err).Errorln("failed to marshal the event data, rejecting event...")
					continue
				}

				logger.Infoln("dispatching event on data channel...")
				channels.Data <- payload
			}
		case err := <-watcher.Errors:
			return errors.Wrapf(err, "failed to watch events for %s", eventSource.Name)
		case <-channels.Done:
			return nil
		}
	}
}
