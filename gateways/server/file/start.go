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

package file

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/gateways/server/common/fsevent"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/fsnotify/fsnotify"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventListener implements Eventing for file event source
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
}

// StartEventSource starts an event source.
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

// listenEvents listen to file related events.
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSourceName, eventSource.Name)

	logger.Infoln("parsing the event source")
	var fileEventSource *v1alpha1.FileEventSource
	if err := yaml.Unmarshal(eventSource.Value, &fileEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
	}

	// create new fs watcher
	logger.Infoln("setting up a new file watcher...")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrapf(err, "failed to set up a file watcher for %s", eventSource.Name)
	}
	defer watcher.Close()

	// file descriptor to watch must be available in file system. You can't watch an fs descriptor that is not present.
	logger.Infoln("adding directory to monitor for the watcher...")
	err = watcher.Add(fileEventSource.WatchPathConfig.Directory)
	if err != nil {
		return errors.Wrapf(err, "failed to add directory %s to the watcher for %s", fileEventSource.WatchPathConfig.Directory, eventSource.Name)
	}

	var pathRegexp *regexp.Regexp
	if fileEventSource.WatchPathConfig.PathRegexp != "" {
		logger.WithField("regex", fileEventSource.WatchPathConfig.PathRegexp).Infoln("matching file path with configured regex...")
		pathRegexp, err = regexp.Compile(fileEventSource.WatchPathConfig.PathRegexp)
		if err != nil {
			return errors.Wrapf(err, "failed to match file path with configured regex %s for %s", fileEventSource.WatchPathConfig.PathRegexp, eventSource.Name)
		}
	}

	logger.Info("listening to file notifications...")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				logger.Info("fs watcher has stopped")
				// watcher stopped watching file events
				return errors.Errorf("fs watcher stopped for %s", eventSource.Name)
			}
			// fwc.Path == event.Name is required because we don't want to send event when .swp files are created
			matched := false
			relPath := strings.TrimPrefix(event.Name, fileEventSource.WatchPathConfig.Directory)
			if fileEventSource.WatchPathConfig.Path != "" && fileEventSource.WatchPathConfig.Path == relPath {
				matched = true
			} else if pathRegexp != nil && pathRegexp.MatchString(relPath) {
				matched = true
			}
			if matched && fileEventSource.EventType == event.Op.String() {
				logger.WithFields(
					map[string]interface{}{
						"event-type":      event.Op.String(),
						"descriptor-name": event.Name,
					},
				).Infoln("file event")

				// Assume fsnotify event has the same Op spec of our file event
				fileEvent := fsevent.Event{Name: event.Name, Op: fsevent.NewOp(event.Op.String())}
				payload, err := json.Marshal(fileEvent)
				if err != nil {
					logger.WithError(err).Errorln("failed to marshal the event to the fs event")
					continue
				}
				logger.WithFields(
					map[string]interface{}{
						"event-type":      event.Op.String(),
						"descriptor-name": event.Name,
					},
				).Infoln("dispatching file event on data channel...")
				channels.Data <- payload
			}
		case err := <-watcher.Errors:
			return errors.Wrapf(err, "failed to process %s", eventSource.Name)
		case <-channels.Done:
			logger.Infoln("event source has been stopped")
			return nil
		}
	}
}
