package main

// fileWatcherConfig contains configuration information for this gateway
// +k8s:openapi-gen=true
type fileWatcherConfig struct {
	// Directory to watch for events
	Directory string
	// Path is relative path of object to watch with respect to the directory
	Path string
	// Type of file operations to watch
	// Refer https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go for more information
	Type string
}
