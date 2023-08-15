package naivewatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/argoproj/argo-events/eventsources/common/fsevent"
	"github.com/stretchr/testify/assert"
)

type WatchableTestFS struct {
}

type TestFSID struct {
	Device int32
	Inode  uint64
}

func (w *WatchableTestFS) Walk(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

func (w *WatchableTestFS) GetFileID(fi os.FileInfo) interface{} {
	stat := fi.Sys().(*syscall.Stat_t)
	return TestFSID{
		Device: int32(stat.Dev),
		Inode:  stat.Ino,
	}
}

func TestWatcherAutoCheck(t *testing.T) {
	watcher, err := NewWatcher(&WatchableTestFS{})
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	tmpdir, err := os.MkdirTemp("", "naive-watcher-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	err = watcher.Add(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	err = watcher.Start(50 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := watcher.Stop(); err != nil {
			fmt.Printf("failed to stop the watcher. err: %+v\n", err)
		}
	}()

	// Create a file
	_, err = os.Create(filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	events := readEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Create, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	// Rename a file
	err = os.Rename(filepath.Join(tmpdir, "foo"), filepath.Join(tmpdir, "bar"))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Rename, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Write a file
	err = os.WriteFile(filepath.Join(tmpdir, "bar"), []byte("wow"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Write, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Chmod a file
	err = os.Chmod(filepath.Join(tmpdir, "bar"), 0777)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Chmod, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Rename & Write & Chmod a file
	err = os.Rename(filepath.Join(tmpdir, "bar"), filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpdir, "foo"), []byte("wowwow"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chmod(filepath.Join(tmpdir, "foo"), 0770)
	if err != nil {
		t.Fatal(err)
	}
	var actualOps fsevent.Op
	time.Sleep(300 * time.Millisecond)
	events = readEvents(t, watcher)
	for _, event := range events {
		if event.Name == filepath.Join(tmpdir, "foo") {
			actualOps |= event.Op
		}
	}
	assert.Equal(t, fsevent.Write|fsevent.Rename|fsevent.Chmod, actualOps)

	// Remove a file
	err = os.Remove(filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Remove, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	err = watcher.Stop()
	if err != nil {
		t.Fatal(err)
	}

	err = watcher.Remove(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatcherManualCheck(t *testing.T) {
	watcher, err := NewWatcher(&WatchableTestFS{})
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	tmpdir, err := os.MkdirTemp("", "naive-watcher-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	err = watcher.Add(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	events := checkAndReadEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{}, events)

	// Create a file
	_, err = os.Create(filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Create, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	// Rename a file
	err = os.Rename(filepath.Join(tmpdir, "foo"), filepath.Join(tmpdir, "bar"))
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Rename, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Write a file
	err = os.WriteFile(filepath.Join(tmpdir, "bar"), []byte("wow"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Write, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Chmod a file
	err = os.Chmod(filepath.Join(tmpdir, "bar"), 0777)
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Chmod, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Rename & Write & Chmod a file
	err = os.Rename(filepath.Join(tmpdir, "bar"), filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpdir, "foo"), []byte("wowwow"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chmod(filepath.Join(tmpdir, "foo"), 0770)
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Write | fsevent.Rename | fsevent.Chmod, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	// Remove a file
	err = os.Remove(filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []fsevent.Event{
		{Op: fsevent.Remove, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	err = watcher.Remove(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
}

func checkAndReadEvents(t *testing.T, watcher *Watcher) []fsevent.Event {
	err := watcher.Check()
	if err != nil {
		t.Fatal(err)
	}
	return readEvents(t, watcher)
}

func readEvents(t *testing.T, watcher *Watcher) []fsevent.Event {
	events := []fsevent.Event{}
L:
	for {
		select {
		case event := <-watcher.Events:
			events = append(events, event)
		default:
			break L
		}
	}
	return events
}
