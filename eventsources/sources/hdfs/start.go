package hdfs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/colinmarc/hdfs"
	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/common/fsevent"
	"github.com/argoproj/argo-events/eventsources/common/naivewatcher"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for HDFS events
type EventListener struct {
	EventSourceName string
	EventName       string
	HDFSEventSource v1alpha1.HDFSEventSource
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.HDFSEvent
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

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	log.Infoln("started processing the Emitter event source...")
	defer sources.Recover(el.GetEventName())

	hdfsEventSource := &el.HDFSEventSource

	log.Infoln("setting up HDFS configuration...")
	hdfsConfig, err := createHDFSConfig(hdfsEventSource)
	if err != nil {
		return errors.Wrapf(err, "failed to create HDFS configuration for %s", el.GetEventName())
	}

	log.Infoln("setting up HDFS client...")
	hdfscli, err := createHDFSClient(hdfsConfig.Addresses, hdfsConfig.HDFSUser, hdfsConfig.KrbOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to create the HDFS client for %s", el.GetEventName())
	}
	defer hdfscli.Close()

	log.Infoln("setting up a new watcher...")
	watcher, err := naivewatcher.NewWatcher(&WatchableHDFS{hdfscli: hdfscli})
	if err != nil {
		return errors.Wrapf(err, "failed to create the HDFS watcher for %s", el.GetEventName())
	}
	defer watcher.Close()

	intervalDuration := 1 * time.Minute
	if hdfsEventSource.CheckInterval != "" {
		d, err := time.ParseDuration(hdfsEventSource.CheckInterval)
		if err != nil {
			return errors.Wrapf(err, "failed to parse the check in interval for %s", el.GetEventName())
		}
		intervalDuration = d
	}

	log.Infoln("started HDFS watcher")
	err = watcher.Start(intervalDuration)
	if err != nil {
		return errors.Wrapf(err, "failed to start the watcher for %s", el.GetEventName())
	}

	// directory to watch must be available in HDFS. You can't watch a directory that is not present.
	log.Infoln("adding configured directory to watcher...")
	err = watcher.Add(hdfsEventSource.Directory)
	if err != nil {
		return errors.Wrapf(err, "failed to add directory %s for %s", hdfsEventSource.Directory, el.GetEventName())
	}

	op := fsevent.NewOp(hdfsEventSource.Type)
	var pathRegexp *regexp.Regexp
	if hdfsEventSource.PathRegexp != "" {
		pathRegexp, err = regexp.Compile(hdfsEventSource.PathRegexp)
		if err != nil {
			return errors.Wrapf(err, "failed to compile the path regex %s for %s", hdfsEventSource.PathRegexp, el.GetEventName())
		}
	}

	log.Infoln("listening to HDFS notifications...")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Info("HDFS watcher has stopped")
				// watcher stopped watching file events
				return errors.Errorf("watcher has been stopped for %s", el.GetEventName())
			}
			matched := false
			relPath := strings.TrimPrefix(event.Name, hdfsEventSource.Directory)

			if hdfsEventSource.Path != "" && hdfsEventSource.Path == relPath {
				matched = true
			} else if pathRegexp != nil && pathRegexp.MatchString(relPath) {
				matched = true
			}

			if matched && (op&event.Op != 0) {
				log = log.WithFields(
					map[string]interface{}{
						"event-type":      event.Op.String(),
						"descriptor-name": event.Name,
					},
				)
				log.Infoln("received an event")

				log.Infoln("parsing the event...")
				payload, err := json.Marshal(event)
				if err != nil {
					log.WithError(err).Errorln("failed to marshal the event data, rejecting event...")
					continue
				}

				log.Infoln("dispatching event on data channel...")
				if err = dispatch(payload); err != nil {
					log.WithError(err).Errorln("failed to dispatch event...")
				}
			}
		case err := <-watcher.Errors:
			return errors.Wrapf(err, "failed to watch events for %s", el.GetEventName())
		case <-stopCh:
			return nil
		}
	}
}
