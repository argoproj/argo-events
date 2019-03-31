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
	"fmt"
	"regexp"
	"strings"

	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/common/fsevent"
	"github.com/fsnotify/fsnotify"
)

// StartEventSource starts an event source
func (ese *FileEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	log := ese.Log.WithEventSource(eventSource.Name)

	log.Info("activating event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(config.(*fileWatcher), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, ese.Log)
}

func (ese *FileEventSourceExecutor) listenEvents(fwc *fileWatcher, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	log := ese.Log.WithEventSource(eventSource.Name)

	// create new fs watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errorCh <- err
		return
	}
	defer watcher.Close()

	// file descriptor to watch must be available in file system. You can't watch an fs descriptor that is not present.
	err = watcher.Add(fwc.Directory)
	if err != nil {
		errorCh <- err
		return
	}

	var pathRegexp *regexp.Regexp
	if fwc.PathRegexp != "" {
		pathRegexp, err = regexp.Compile(fwc.PathRegexp)
		if err != nil {
			errorCh <- err
			return
		}
	}

	log.Info("starting to watch to file notifications")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Info("fs watcher has stopped")
				// watcher stopped watching file events
				errorCh <- fmt.Errorf("fs watcher stopped")
				return
			}
			// fwc.Path == event.Name is required because we don't want to send event when .swp files are created
			matched := false
			relPath := strings.TrimPrefix(event.Name, fwc.Directory)
			if fwc.Path != "" && fwc.Path == relPath {
				matched = true
			} else if pathRegexp != nil && pathRegexp.MatchString(relPath) {
				matched = true
			}
			if matched && fwc.Type == event.Op.String() {
				log.WithFields(
					map[string]interface{}{
						"event-type":      event.Op.String(),
						"descriptor-name": event.Name,
					},
				).Debug("fs event")

				// Assume fsnotify event has the same Op spec of our file event
				fileEvent := fsevent.Event{Name: event.Name, Op: fsevent.NewOp(event.Op.String())}
				payload, err := json.Marshal(fileEvent)
				if err != nil {
					errorCh <- err
					return
				}
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
