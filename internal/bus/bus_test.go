package bus_test

import (
	"testing"

	"github.com/brazier/brazier/internal/bus"
)

func TestPublishSubscribe(t *testing.T) {
	b := bus.New()
	var got []bus.Event

	b.Subscribe(func(e bus.Event) { got = append(got, e) },
		bus.EventJobCompleted, bus.EventRunCompleted)

	b.Publish(bus.Event{Type: bus.EventJobCompleted, Payload: "job-1"})
	b.Publish(bus.Event{Type: bus.EventTriggerReceived, Payload: "ignored"})
	b.Publish(bus.Event{Type: bus.EventRunCompleted, Payload: "run-1"})

	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].Type != bus.EventJobCompleted {
		t.Errorf("got[0].Type = %q", got[0].Type)
	}
	if got[1].Type != bus.EventRunCompleted {
		t.Errorf("got[1].Type = %q", got[1].Type)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	b := bus.New()
	count := 0
	b.Subscribe(func(bus.Event) { count++ }, bus.EventJobDispatched)
	b.Subscribe(func(bus.Event) { count++ }, bus.EventJobDispatched)
	b.Publish(bus.Event{Type: bus.EventJobDispatched})
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}
