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
	"fmt"
	"strings"

	"github.com/argoproj/argo-events/gateways"
	"github.com/fsnotify/fsnotify"
)

// StartEventSource starts an event source
func (ese *FileEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("activating event source")
	f, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(f, eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
}

func (ese *FileEventSourceExecutor) listenEvents(fwc *fileWatcher, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

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

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("starting to watch to file notifications")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				ese.Log.Info().Str("event-source", eventSource.Name).Msg("fs watcher has stopped")
				// watcher stopped watching file events
				errorCh <- fmt.Errorf("fs watcher stopped")
				return
			}
			// fwc.Path == event.Name is required because we don't want to send event when .swp files are created
			if fwc.Path == strings.TrimPrefix(event.Name, fwc.Directory) && fwc.Type == event.Op.String() {
				ese.Log.Debug().Str("config-key", eventSource.Name).Str("event-type", event.Op.String()).Str("descriptor-name", event.Name).Msg("fs event")
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
