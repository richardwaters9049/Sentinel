package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type NATS struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

func Connect(ctx context.Context, url string, timeout time.Duration) (*NATS, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	conn, err := nats.Connect(
		url,
		nats.Name("sentinel-gateway"),
		nats.Timeout(timeout),
		nats.MaxReconnects(3),
		nats.ReconnectWait(500*time.Millisecond),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("initialise JetStream: %w", err)
	}

	accountInfoDone := make(chan error, 1)
	go func() {
		_, infoErr := js.AccountInfo()
		accountInfoDone <- infoErr
	}()

	select {
	case <-ctx.Done():
		conn.Close()
		return nil, fmt.Errorf("verify JetStream: %w", ctx.Err())
	case <-time.After(timeout):
		conn.Close()
		return nil, fmt.Errorf("verify JetStream: timeout after %s", timeout)
	case infoErr := <-accountInfoDone:
		if infoErr != nil {
			conn.Close()
			return nil, fmt.Errorf("verify JetStream: %w", infoErr)
		}
	}

	return &NATS{conn: conn, js: js}, nil
}

func (n *NATS) Ping(ctx context.Context) error {
	if n == nil || n.conn == nil {
		return fmt.Errorf("NATS is not initialised")
	}

	if !n.conn.IsConnected() {
		return fmt.Errorf("NATS connection is not active")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := n.conn.FlushTimeout(2 * time.Second); err != nil {
		return fmt.Errorf("flush NATS connection: %w", err)
	}

	return nil
}

func (n *NATS) JetStream() nats.JetStreamContext {
	return n.js
}

func (n *NATS) Close() {
	if n == nil || n.conn == nil {
		return
	}

	_ = n.conn.Drain()
	n.conn.Close()
}
