// watcher.go watches the configured board column directories for external file changes
// (e.g. the user editing a ticket in their editor while the TUI is open) and
// sends a signal on a channel so the TUI can reload the board.
//
// Debouncing: rapid filesystem events (common during saves) are collapsed into
// a single notification by resetting a timer on each event. Only after the
// debounce period has elapsed with no further events is the signal sent.
// The channel is buffered with capacity 1; if a signal is already pending the
// new event is silently dropped rather than blocking the watcher goroutine.
package tui

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/dawidsok/tickcats/internal/store"
)

type fileWatcher struct {
	ch      chan struct{}
	watcher *fsnotify.Watcher
}

func newFileWatcher(root string) (*fileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	cfg, err := store.LoadConfig(root)
	if err != nil {
		cfg = store.Config{}
	}
	for _, col := range cfg.GetColumns() {
		_ = w.Add(filepath.Join(root, col.ID))
	}
	fw := &fileWatcher{
		ch:      make(chan struct{}, 1),
		watcher: w,
	}
	go fw.run(300 * time.Millisecond)
	return fw, nil
}

func (fw *fileWatcher) run(debounce time.Duration) {
	var timer *time.Timer
	for {
		select {
		case _, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, func() {
				select {
				case fw.ch <- struct{}{}:
				default:
				}
			})
		case _, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
