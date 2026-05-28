package watch

import "github.com/gpupaas-ai/gpupaas-go/runtime"

// EventType describes a watch event.
type EventType string

const (
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
	Error    EventType = "ERROR"
)

// Event is a watch notification.
type Event struct {
	Type   EventType      `json:"type"`
	Object runtime.Object `json:"object"`
}

// Interface receives watch events.
type Interface interface {
	Stop()
	ResultChan() <-chan Event
}

// FakeWatcher is a minimal watcher for tests.
type FakeWatcher struct {
	events chan Event
	stop   chan struct{}
}

// NewFakeWatcher creates a watcher with buffered events.
func NewFakeWatcher(events ...Event) *FakeWatcher {
	ch := make(chan Event, len(events))
	for _, e := range events {
		ch <- e
	}
	close(ch)
	return &FakeWatcher{events: ch, stop: make(chan struct{})}
}

func (f *FakeWatcher) Stop() {
	select {
	case <-f.stop:
	default:
		close(f.stop)
	}
}

func (f *FakeWatcher) ResultChan() <-chan Event {
	return f.events
}
