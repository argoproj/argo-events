package naivewatcher

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/argoproj/argo-events/gateways/common/fileevent"
)

const (
	eventsQueueSize = 100
	errorsQueueSize = 100
)

// Watchable is an interface of FS which can be watched by this watcher
type Watchable interface {
	Walk(root string, walkFn filepath.WalkFunc) error
	GetFileID(fi os.FileInfo) interface{}
}

// filePathAndInfo is an internal struct to manage path and file info
type filePathAndInfo struct {
	path  string
	info  os.FileInfo
	found bool
}

// Watcher is a file watcher
type Watcher struct {
	watchable Watchable

	// manage target directories
	watchList  map[string]map[interface{}]*filePathAndInfo
	mWatchList *sync.RWMutex

	// internal use
	mCheck       *Mutex
	mutexRunning *Mutex
	Events       chan fileevent.Event
	Errors       chan error
	stop         chan struct{}
	stopResp     chan struct{}
}

// NewWatcher is the initializer of watcher struct
func NewWatcher(watchable Watchable) (*Watcher, error) {
	watcher := &Watcher{
		watchable:    watchable,
		mWatchList:   new(sync.RWMutex),
		watchList:    map[string]map[interface{}]*filePathAndInfo{},
		mCheck:       new(Mutex),
		mutexRunning: new(Mutex),
		Events:       make(chan fileevent.Event, eventsQueueSize),
		Errors:       make(chan error, errorsQueueSize),
		stop:         make(chan struct{}),
		stopResp:     make(chan struct{}),
	}
	return watcher, nil
}

// Add adds a directory into the watch list
func (w *Watcher) Add(dir string) error {
	w.mWatchList.Lock()
	defer w.mWatchList.Unlock()
	w.watchList[dir] = nil
	return nil
}

// Remove removes a directory from the watch list
func (w *Watcher) Remove(dir string) error {
	w.mWatchList.Lock()
	defer w.mWatchList.Unlock()
	delete(w.watchList, dir)
	return nil
}

// WatchList returns target directories
func (w *Watcher) WatchList() []string {
	w.mWatchList.RLock()
	defer w.mWatchList.RUnlock()
	dirs := []string{}
	for dir := range w.watchList {
		dirs = append(dirs, dir)
	}
	return dirs
}

// Close cleans up the watcher
func (w *Watcher) Close() error {
	_ = w.Stop()
	close(w.Events)
	close(w.Errors)
	return nil
}

// Start starts the watcher
func (w *Watcher) Start(interval time.Duration) error {
	if !w.mutexRunning.TryLock() {
		return errors.New("watcher has already started")
	}
	// run initial check
	err := w.Check()
	if err != nil {
		return err
	}
	go func() {
		defer w.mutexRunning.Unlock()
		defer close(w.stopResp)
		for {
			select {
			case <-time.After(interval):
				err := w.Check()
				if err != nil {
					w.Errors <- err
				}
			case <-w.stop:
				return
			}
		}
	}()
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	if !w.mutexRunning.IsLocked() {
		return errors.New("watcher is not started")
	}
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.stopResp
	return nil
}

// Check checks the state of target directories
func (w *Watcher) Check() error {
	if !w.mCheck.TryLock() {
		return errors.New("another check is still running")
	}
	defer w.mCheck.Unlock()

	w.mWatchList.Lock()
	defer w.mWatchList.Unlock()

	for dir, walkedFiles := range w.watchList {
		if walkedFiles == nil {
			walkedFiles = make(map[interface{}]*filePathAndInfo)
			w.watchList[dir] = walkedFiles
		}
		for fileID := range walkedFiles {
			walkedFiles[fileID].found = false
		}
		walkFn := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				w.Errors <- err
				return nil
			}
			// Ignore walk root
			if dir == path {
				return nil
			}
			fileID := w.watchable.GetFileID(info)
			lastPathAndInfo := walkedFiles[fileID]
			if lastPathAndInfo == nil {
				w.Events <- fileevent.Event{Op: fileevent.Create, Name: path}
				walkedFiles[fileID] = &filePathAndInfo{path, info, true}
			} else {
				lastInfo := lastPathAndInfo.info
				var op fileevent.Op
				if path != lastPathAndInfo.path {
					op |= fileevent.Rename
				}
				if !info.ModTime().Equal(lastInfo.ModTime()) || info.Size() != lastInfo.Size() {
					op |= fileevent.Write
				}
				if info.Mode() != lastInfo.Mode() {
					op |= fileevent.Chmod
				}
				if op != 0 {
					w.Events <- fileevent.Event{Op: op, Name: path}
				}
				walkedFiles[fileID].path = path
				walkedFiles[fileID].info = info
				walkedFiles[fileID].found = true
			}
			return nil
		}

		err := w.watchable.Walk(dir, walkFn)
		if err != nil {
			return err
		}

		for fileID, pathAndInfo := range walkedFiles {
			if !pathAndInfo.found {
				w.Events <- fileevent.Event{Op: fileevent.Remove, Name: pathAndInfo.path}
				delete(walkedFiles, fileID)
			}
		}
	}

	return nil
}
