// Package bus provides the in-process event bus used by the master service.
// All communication between subsystems is via typed events over Go channels —
// no external broker is required.
package bus

import "sync"

// EventType identifies the kind of event.
type EventType string

const (
	EventTriggerReceived EventType = "TriggerReceived"
	EventJobDispatched   EventType = "JobDispatched"
	EventJobCompleted    EventType = "JobCompleted"
	EventJobFailed       EventType = "JobFailed"
	EventRunCompleted    EventType = "RunCompleted"
)

// Event carries a type and an opaque payload.
type Event struct {
	Type    EventType
	Payload any
}

// Handler is a function that processes an event.
type Handler func(Event)

// Bus is a simple in-process pub/sub bus.
type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

// New returns a ready-to-use Bus.
func New() *Bus {
	return &Bus{handlers: make(map[EventType][]Handler)}
}

// Subscribe registers h to receive events of the given types.
func (b *Bus) Subscribe(h Handler, types ...EventType) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, t := range types {
		b.handlers[t] = append(b.handlers[t], h)
	}
}

// Publish delivers e to all subscribers registered for e.Type.
// Handlers are called synchronously in registration order.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	hs := b.handlers[e.Type]
	b.mu.RUnlock()
	for _, h := range hs {
		h(e)
	}
}
