package events

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// NATSBus implements event bus using NATS JetStream
type NATSBus struct {
	conn *nats.Conn
}

// NewNATSBus creates a new NATS-based event bus
func NewNATSBus(url string) (*NATSBus, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSBus{
		conn: conn,
	}, nil
}

// Start begins processing events
func (b *NATSBus) Start(ctx context.Context) error {
	// TODO: Implement event processing loop
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
