package fsevent

import (
	"errors"
	"path"
	"regexp"
)

type WatchPathConfig struct {
	// Directory to watch for events
	Directory string `json:"directory"`
	// Path is relative path of object to watch with respect to the directory
	Path string `json:"path,omitempty"`
	// PathRegexp is regexp of relative path of object to watch with respect to the directory
	PathRegexp string `json:"pathRegexp,omitempty"`
}

// Validate validates WatchPathConfig
func (c *WatchPathConfig) Validate() error {
	if c.Directory == "" {
		return errors.New("directory is required")
	}
	if !path.IsAbs(c.Directory) {
		return errors.New("directory must be an absolute file path")
	}
	if c.Path == "" && c.PathRegexp == "" {
		return errors.New("either path or pathRegexp must be specified")
	}
	if c.Path != "" && c.PathRegexp != "" {
		return errors.New("path and pathRegexp cannot be specified together")
	}
	if c.Path != "" && path.IsAbs(c.Path) {
		return errors.New("path must be a relative file path")
	}
	if c.PathRegexp != "" {
		_, err := regexp.Compile(c.PathRegexp)
		if err != nil {
			return err
		}
	}
	return nil
}
