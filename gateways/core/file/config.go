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
	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
)

// FileEventSourceExecutor implements ConfigExecutor interface
type FileEventSourceExecutor struct {
	Log zerolog.Logger
}

// FileWatcherConfig contains configuration information for this gateway
// +k8s:openapi-gen=true
type FileWatcherConfig struct {
	// Directory to watch for events
	Directory string `json:"directory"`
	// Path is relative path of object to watch with respect to the directory
	Path string `json:"path"`
	// Type of file operations to watch
	// Refer https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go for more information
	Type string `json:"type"`
}

func parseEventSource(eventSource *string) (*FileWatcherConfig, error) {
	var f *FileWatcherConfig
	err := yaml.Unmarshal([]byte(*eventSource), &f)
	if err != nil {
		return nil, err
	}
	return f, err
}
