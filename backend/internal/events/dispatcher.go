package events

import (
	"context"
	"log/slog"
	"sync"
)

// Event is the envelope for internal domain events.
type Event struct {
	Type    string
	Payload any
}

// Handler functions process an incoming event.
type Handler func(ctx context.Context, e Event) error

// EventDispatcher defines the publish-subscribe mechanism for domain events.
type EventDispatcher interface {
	Subscribe(eventType string, handler Handler)
	Publish(ctx context.Context, e Event)
}

// asyncDispatcher provides a simple in-memory implementation of EventDispatcher.
type asyncDispatcher struct {
	handlers map[string][]Handler
	mu       sync.RWMutex
}

// NewAsyncDispatcher creates a new EventDispatcher.
func NewAsyncDispatcher() EventDispatcher {
	return &asyncDispatcher{
		handlers: make(map[string][]Handler),
	}
}

func (d *asyncDispatcher) Subscribe(eventType string, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

func (d *asyncDispatcher) Publish(ctx context.Context, e Event) {
	d.mu.RLock()
	handlers, ok := d.handlers[e.Type]
	d.mu.RUnlock()

	if !ok {
		return
	}

	// Dispatch to each handler asynchronously
	for _, h := range handlers {
		go func(handler Handler) {
			if err := handler(ctx, e); err != nil {
				slog.ErrorContext(ctx, "failed to handle event", "event_type", e.Type, "error", err)
			}
		}(h)
	}
}
