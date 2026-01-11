package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
)

// NATSBus implements event bus using NATS JetStream
type NATSBus struct {
	conn     *nats.Conn
	js       nats.JetStreamContext
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
}

// NewNATSBus creates a new NATS-based event bus
func NewNATSBus(url string) (*NATSBus, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create stream for events
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"events.>"},
		Storage:  nats.FileStorage,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		conn.Close()
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	return &NATSBus{
		conn:     conn,
		js:       js,
		handlers: make(map[EventType][]EventHandler),
	}, nil
}

// Publish publishes an event to NATS
func (b *NATSBus) Publish(ctx context.Context, event *Event) error {
	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to NATS subject: events.{event_type}
	subject := fmt.Sprintf("events.%s", event.Type)
	_, err = b.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("Published event: %s (ID: %s)", event.Type, event.ID)
	return nil
}

// Subscribe registers a handler for an event type
func (b *NATSBus) Subscribe(eventType EventType, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	return nil
}

// Start begins processing events
func (b *NATSBus) Start(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Subscribe to all registered event types
	for eventType := range b.handlers {
		subject := fmt.Sprintf("events.%s", eventType)

		// Create durable consumer
		_, err := b.js.Subscribe(subject, func(msg *nats.Msg) {
			// Parse event
			var event Event
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				msg.Nak() // Negative acknowledge
				return
			}

			// Call all handlers for this event type
			b.mu.RLock()
			handlers := b.handlers[event.Type]
			b.mu.RUnlock()

			for _, handler := range handlers {
				if err := handler(ctx, &event); err != nil {
					log.Printf("Event handler error for %s: %v", event.Type, err)
					// Don't nak on handler errors, just log
				}
			}

			// Acknowledge message
			msg.Ack()
		}, nats.Durable(fmt.Sprintf("%s-consumer", eventType)))

		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
		}

		log.Printf("Subscribed to event type: %s", eventType)
	}

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

// Close closes the NATS connection
func (b *NATSBus) Close() error {
	if b.conn != nil {
		b.conn.Close()
	}
	return nil
}

