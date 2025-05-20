/*
Copyright 2018 The Argoproj Authors.

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
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	watcherpkg "github.com/radovskyb/watcher"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/fsevent"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

// EventListener implements Eventing for file event source
type EventListener struct {
	EventSourceName string
	EventName       string
	FileEventSource v1alpha1.FileEventSource
	Metrics         *metrics.Metrics
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
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.FileEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	fileEventSource := &el.FileEventSource
	if fileEventSource.Polling {
		if err := el.listenEventsPolling(ctx, dispatch, log); err != nil {
			log.Error("failed to listen to events", zap.Error(err))
			return err
		}
	} else {
		if err := el.listenEvents(ctx, dispatch, log); err != nil {
			log.Error("failed to listen to events", zap.Error(err))
			return err
		}
	}
	return nil
}

// listenEvents listen to file related events.
func (el *EventListener) listenEvents(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
	fileEventSource := &el.FileEventSource

	// create new fs watcher
	log.Info("setting up a new file watcher...")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to set up a file watcher for %s, %w", el.GetEventName(), err)
	}
	defer watcher.Close()

	// file descriptor to watch must be available in file system. You can't watch an fs descriptor that is not present.
	log.Info("adding directory to monitor for the watcher...")
	err = watcher.Add(fileEventSource.WatchPathConfig.Directory)
	if err != nil {
		return fmt.Errorf("failed to add directory %s to the watcher for %s, %w", fileEventSource.WatchPathConfig.Directory, el.GetEventName(), err)
	}

	var pathRegexp *regexp.Regexp
	if fileEventSource.WatchPathConfig.PathRegexp != "" {
		log.Infow("matching file path with configured regex...", zap.Any("regex", fileEventSource.WatchPathConfig.PathRegexp))
		pathRegexp, err = regexp.Compile(fileEventSource.WatchPathConfig.PathRegexp)
		if err != nil {
			return fmt.Errorf("failed to match file path with configured regex %s for %s, %w", fileEventSource.WatchPathConfig.PathRegexp, el.GetEventName(), err)
		}
	}

	processOne := func(event fsnotify.Event) error {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		log.Infow("file event", zap.Any("event-type", event.Op.String()), zap.Any("descriptor-name", event.Name))

		// Assume fsnotify event has the same Op spec of our file event
		fileEvent := fsevent.Event{Name: event.Name, Op: fsevent.NewOp(event.Op.String()), Metadata: el.FileEventSource.Metadata}
		payload, err := json.Marshal(fileEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal the event to the fs event, %w", err)
		}
		log.Infow("dispatching file event on data channel...", zap.Any("event-type", event.Op.String()), zap.Any("descriptor-name", event.Name))
		if err = dispatch(payload); err != nil {
			return fmt.Errorf("failed to dispatch a file event, %w", err)
		}
		return nil
	}

	log.Info("listening to file notifications...")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Info("fs watcher has stopped")
				// watcher stopped watching file events
				return fmt.Errorf("fs watcher stopped for %s", el.GetEventName())
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
				if err = processOne(event); err != nil {
					log.Errorw("failed to process a file event", zap.Error(err))
					el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
				}
			}
		case err := <-watcher.Errors:
			return fmt.Errorf("failed to process %s, %w", el.GetEventName(), err)
		case <-ctx.Done():
			log.Info("event source has been stopped")
			return nil
		}
	}
}

// listenEvents listen to file related events using polling.
func (el *EventListener) listenEventsPolling(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
	fileEventSource := &el.FileEventSource

	// create new fs watcher
	log.Info("setting up a new file polling watcher...")
	watcher := watcherpkg.New()
	defer watcher.Close()

	// file descriptor to watch must be available in file system. You can't watch an fs descriptor that is not present.
	log.Info("adding directory to monitor for the watcher...")
	err := watcher.Add(fileEventSource.WatchPathConfig.Directory)
	if err != nil {
		return fmt.Errorf("failed to add directory %s to the watcher for %s, %w", fileEventSource.WatchPathConfig.Directory, el.GetEventName(), err)
	}

	var pathRegexp *regexp.Regexp
	if fileEventSource.WatchPathConfig.PathRegexp != "" {
		log.Infow("matching file path with configured regex...", zap.Any("regex", fileEventSource.WatchPathConfig.PathRegexp))
		pathRegexp, err = regexp.Compile(fileEventSource.WatchPathConfig.PathRegexp)
		if err != nil {
			return fmt.Errorf("failed to match file path with configured regex %s for %s, %w", fileEventSource.WatchPathConfig.PathRegexp, el.GetEventName(), err)
		}
	}

	processOne := func(event watcherpkg.Event) error {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		log.Infow("file event", zap.Any("event-type", event.Op.String()), zap.Any("descriptor-name", event.Name))

		// Assume fsnotify event has the same Op spec of our file event
		fileEvent := fsevent.Event{Name: event.Name(), Op: fsevent.NewOp(event.Op.String()), Metadata: el.FileEventSource.Metadata}
		payload, err := json.Marshal(fileEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal the event to the fs event, %w", err)
		}
		log.Infow("dispatching file event on data channel...", zap.Any("event-type", event.Op.String()), zap.Any("descriptor-name", event.Name))
		if err = dispatch(payload); err != nil {
			return fmt.Errorf("failed to dispatch file event, %w", err)
		}
		return nil
	}

	go func() {
		log.Info("listening to file notifications...")
		for {
			select {
			case event, ok := <-watcher.Event:
				if !ok {
					log.Info("fs watcher has stopped")
					// watcher stopped watching file events
					log.Errorw("fs watcher stopped", zap.Any("eventName", el.GetEventName()))
					return
				}
				// fwc.Path == event.Name is required because we don't want to send event when .swp files are created
				matched := false
				relPath := strings.TrimPrefix(event.Name(), fileEventSource.WatchPathConfig.Directory)
				if fileEventSource.WatchPathConfig.Path != "" && fileEventSource.WatchPathConfig.Path == relPath {
					matched = true
				} else if pathRegexp != nil && pathRegexp.MatchString(relPath) {
					matched = true
				}
				if matched && fileEventSource.EventType == event.Op.String() {
					if err := processOne(event); err != nil {
						log.Errorw("failed to process a file event", zap.Error(err))
						el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
					}
				}
			case err := <-watcher.Error:
				log.Errorw("failed to process event source", zap.Any("eventName", el.GetEventName()), zap.Error(err))
				return
			case <-ctx.Done():
				log.Info("event source has been stopped")
				return
			}
		}
	}()
	log.Info("Starting watcher...")
	if err = watcher.Start(time.Millisecond * 100); err != nil {
		return fmt.Errorf("failed to start watcher for %s, %w", el.GetEventName(), err)
	}
	return nil
}
