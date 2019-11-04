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

// EventSourceListener implements Eventing for file event source
type EventSourceListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
}

// StartEventSource starts an event source
func (listener *EventSourceListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	log := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("started processing event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)
	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents listen to file related events
func (listener *EventSourceListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer server.Recover(eventSource.Name)

	var fileEventSource *v1alpha1.FileEventSource
	if err := yaml.Unmarshal(eventSource.Value, &fileEventSource); err != nil {
		errorCh <- err
		return
	}

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	// create new fs watcher
	logger.Infoln("setting up a new file watcher...")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errorCh <- err
		return
	}
	defer watcher.Close()

	// file descriptor to watch must be available in file system. You can't watch an fs descriptor that is not present.
	logger.Infoln("adding directory to monitor for the watcher...")
	err = watcher.Add(fileEventSource.Directory)
	if err != nil {
		errorCh <- err
		return
	}

	var pathRegexp *regexp.Regexp
	if fileEventSource.PathRegexp != "" {
		logger.WithField("regex", fileEventSource.PathRegexp).Infoln("matching file path with configured regex...")
		pathRegexp, err = regexp.Compile(fileEventSource.PathRegexp)
		if err != nil {
			errorCh <- err
			return
		}
	}

	logger.Info("listening to file notifications...")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				logger.Info("fs watcher has stopped")
				// watcher stopped watching file events
				errorCh <- errors.New("fs watcher stopped")
				return
			}
			// fwc.Path == event.Name is required because we don't want to send event when .swp files are created
			matched := false
			relPath := strings.TrimPrefix(event.Name, fileEventSource.Directory)
			if fileEventSource.Path != "" && fileEventSource.Path == relPath {
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
					errorCh <- err
					return
				}
				logger.WithFields(
					map[string]interface{}{
						"event-type":      event.Op.String(),
						"descriptor-name": event.Name,
					},
				).Infoln("dispatching file event on data channel...")
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
