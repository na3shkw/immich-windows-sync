package watcher

import (
	"context"

	"github.com/fsnotify/fsnotify"
)

type EventType int

const (
	Unknown EventType = iota
	Create
	Write
	Remove
)

type Event struct {
	Type EventType
	Path string
}

type Watcher struct {
	Events  chan Event
	cancel  context.CancelFunc
	watcher *fsnotify.Watcher
}
