package naivewatcher

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

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
		Device: stat.Dev,
		Inode:  stat.Ino,
	}
}

func TestWatcherAutoCheck(t *testing.T) {
	watcher, err := NewWatcher(&WatchableTestFS{})
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	tmpdir, err := ioutil.TempDir("", "naive-watcher-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	err = watcher.Add(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	err = watcher.Start(100 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Stop()

	// Create a file
	_, err = os.Create(filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	events := readEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Create, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	// Rename a file
	err = os.Rename(filepath.Join(tmpdir, "foo"), filepath.Join(tmpdir, "bar"))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Rename, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Write a file
	err = ioutil.WriteFile(filepath.Join(tmpdir, "bar"), []byte("wow"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Write, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Chmod a file
	err = os.Chmod(filepath.Join(tmpdir, "bar"), 0777)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Chmod, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Rename & Write & Chmod a file
	err = os.Rename(filepath.Join(tmpdir, "bar"), filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(tmpdir, "foo"), []byte("wowwow"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chmod(filepath.Join(tmpdir, "foo"), 0770)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Write | Rename | Chmod, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	// Remove a file
	err = os.Remove(filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	events = readEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Remove, Name: filepath.Join(tmpdir, "foo")},
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

	tmpdir, err := ioutil.TempDir("", "naive-watcher-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	err = watcher.Add(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	events := checkAndReadEvents(t, watcher)
	assert.Equal(t, []Event{}, events)

	// Create a file
	_, err = os.Create(filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Create, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	// Rename a file
	err = os.Rename(filepath.Join(tmpdir, "foo"), filepath.Join(tmpdir, "bar"))
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Rename, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Write a file
	err = ioutil.WriteFile(filepath.Join(tmpdir, "bar"), []byte("wow"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Write, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Chmod a file
	err = os.Chmod(filepath.Join(tmpdir, "bar"), 0777)
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Chmod, Name: filepath.Join(tmpdir, "bar")},
	}, events)

	// Rename & Write & Chmod a file
	err = os.Rename(filepath.Join(tmpdir, "bar"), filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(tmpdir, "foo"), []byte("wowwow"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chmod(filepath.Join(tmpdir, "foo"), 0770)
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Write | Rename | Chmod, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	// Remove a file
	err = os.Remove(filepath.Join(tmpdir, "foo"))
	if err != nil {
		t.Fatal(err)
	}
	events = checkAndReadEvents(t, watcher)
	assert.Equal(t, []Event{
		{Op: Remove, Name: filepath.Join(tmpdir, "foo")},
	}, events)

	err = watcher.Remove(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
}

func checkAndReadEvents(t *testing.T, watcher *Watcher) []Event {
	err := watcher.Check()
	if err != nil {
		t.Fatal(err)
	}
	return readEvents(t, watcher)
}

func readEvents(t *testing.T, watcher *Watcher) []Event {
	events := []Event{}
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
