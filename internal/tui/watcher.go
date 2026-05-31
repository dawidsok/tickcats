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
	for _, state := range store.ValidStates {
		_ = w.Add(filepath.Join(root, string(state)))
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
