// Package watcher watches the grouping folders (not the projects, so it does not
// react to every file save) and reports changes with a debounce.
package watcher

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	mu       sync.Mutex
	w        *fsnotify.Watcher
	watched  map[string]bool
	onChange func()
	timer    *time.Timer
	delay    time.Duration
	done     chan struct{}
}

// New starts the watcher; onChange is called at most once every delay.
func New(delay time.Duration, onChange func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{w: fw, watched: map[string]bool{}, onChange: onChange, delay: delay, done: make(chan struct{})}
	go w.loop()
	return w, nil
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.w.Events:
			if !ok {
				return
			}
			// content-only changes do not change the tree
			if ev.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				w.schedule()
			}
		case _, ok := <-w.w.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) schedule() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.delay, w.onChange)
}

// Set replaces the set of watched folders (non-recursive).
func (w *Watcher) Set(dirs []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	next := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		next[d] = true
	}
	for d := range w.watched {
		if !next[d] {
			_ = w.w.Remove(d)
			delete(w.watched, d)
		}
	}
	for d := range next {
		if !w.watched[d] {
			if err := w.w.Add(d); err == nil {
				w.watched[d] = true
			}
		}
	}
}

func (w *Watcher) Close() error {
	close(w.done)
	return w.w.Close()
}
