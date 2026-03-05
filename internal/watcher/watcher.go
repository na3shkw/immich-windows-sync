package watcher

import (
	"context"

	"github.com/fsnotify/fsnotify"
)

func NewWatcher() (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		Events:  make(chan Event),
		Errors:  make(chan error),
		cancel:  nil,
		watcher: watcher,
	}, nil
}

func (w *Watcher) Start(targetDirs []string) error {
	for _, v := range targetDirs {
		err := w.watcher.Add(v)
		if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go func() {
		for {
			select {
			case fsnotifyEvent, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				w.Events <- Event{
					Type: toEventType(fsnotifyEvent),
					Path: fsnotifyEvent.Name,
				}
			case err := <-w.watcher.Errors:
				w.Errors <- err
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (w *Watcher) Stop() error {
	w.cancel()
	err := w.watcher.Close()
	if err != nil {
		return err
	}
	return nil
}

func toEventType(event fsnotify.Event) EventType {
	if event.Op.Has(fsnotify.Create) {
		return Create
	}
	if event.Op.Has(fsnotify.Write) {
		return Write
	}
	if event.Op.Has(fsnotify.Remove) {
		return Remove
	}
	if event.Op.Has(fsnotify.Rename) {
		return Rename
	}
	return Unknown
}
