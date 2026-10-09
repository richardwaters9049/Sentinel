package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	TelemetryStream       = "SENTINEL_TELEMETRY"
	TelemetrySubject      = "sentinel.telemetry.v1"
	TelemetryStoreDurable = "sentinel-event-store-workers-v1"
)

type TelemetryHandler func(context.Context, []byte) error
type TelemetryErrorHandler func(error)

type permanentFailure interface {
	Permanent() bool
}

type NATS struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// ConsumerStats exposes backlog without reading event payloads.
func (n *NATS) ConsumerStats(ctx context.Context) (uint64, int, error) {
	info, err := n.js.ConsumerInfo(TelemetryStream, TelemetryStoreDurable, nats.Context(ctx))
	if err != nil {
		return 0, 0, err
	}
	return info.NumPending, info.NumAckPending, nil
}

func Connect(ctx context.Context, url string, timeout time.Duration) (*NATS, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	conn, err := nats.Connect(
		url,
		nats.Name("sentinel-gateway"),
		nats.Timeout(timeout),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
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

	client := &NATS{conn: conn, js: js}
	if err := client.ensureTelemetryStream(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

func (n *NATS) ensureTelemetryStream() error {
	config := &nats.StreamConfig{
		Name:        TelemetryStream,
		Description: "Normalised Sentinel security telemetry",
		Subjects:    []string{TelemetrySubject},
		Retention:   nats.LimitsPolicy,
		Storage:     nats.FileStorage,
		Discard:     nats.DiscardOld,
		MaxAge:      24 * time.Hour,
		Duplicates:  2 * time.Minute,
	}

	info, err := n.js.StreamInfo(TelemetryStream)
	if err != nil {
		if err == nats.ErrStreamNotFound {
			if _, addErr := n.js.AddStream(config); addErr != nil {
				return fmt.Errorf("create telemetry stream: %w", addErr)
			}
			return nil
		}
		return fmt.Errorf("inspect telemetry stream: %w", err)
	}

	config.MaxMsgs = info.Config.MaxMsgs
	config.MaxBytes = info.Config.MaxBytes
	config.MaxMsgSize = info.Config.MaxMsgSize
	config.Replicas = info.Config.Replicas

	if _, err := n.js.UpdateStream(config); err != nil {
		return fmt.Errorf("update telemetry stream: %w", err)
	}

	return nil
}

func (n *NATS) PublishTelemetry(ctx context.Context, eventID string, payload []byte) error {
	if n == nil || n.conn == nil || n.js == nil {
		return fmt.Errorf("NATS is not initialised")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	ack, err := n.js.Publish(
		TelemetrySubject,
		payload,
		nats.MsgId(eventID),
	)
	if err != nil {
		return fmt.Errorf("publish telemetry: %w", err)
	}
	if ack.Stream != TelemetryStream {
		return fmt.Errorf("publish telemetry: unexpected stream %q", ack.Stream)
	}

	return nil
}

func (n *NATS) StartTelemetryConsumer(
	ctx context.Context,
	handler TelemetryHandler,
	onError TelemetryErrorHandler,
) (*nats.Subscription, error) {
	if n == nil || n.conn == nil || n.js == nil {
		return nil, fmt.Errorf("NATS is not initialised")
	}
	if handler == nil {
		return nil, fmt.Errorf("telemetry handler is required")
	}

	notifyError := func(err error) {
		if err != nil && onError != nil {
			onError(err)
		}
	}

	subscription, err := n.js.QueueSubscribe(
		TelemetrySubject,
		"sentinel-event-store-workers",
		func(message *nats.Msg) {
			if ctx.Err() != nil {
				notifyError(message.Nak())
				return
			}

			handlerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := handler(handlerCtx, message.Data)
			cancel()

			if err != nil {
				notifyError(err)

				var permanent permanentFailure
				if errors.As(err, &permanent) && permanent.Permanent() {
					notifyError(message.Term())
					return
				}

				notifyError(message.NakWithDelay(time.Second))
				return
			}

			notifyError(message.Ack())
		},
		nats.Durable(TelemetryStoreDurable),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.AckWait(15*time.Second),
		nats.MaxDeliver(-1),
		nats.DeliverNew(),
		nats.BindStream(TelemetryStream),
	)
	if err != nil {
		return nil, fmt.Errorf("start telemetry consumer: %w", err)
	}

	return subscription, nil
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

func (n *NATS) Close() {
	if n == nil || n.conn == nil {
		return
	}

	_ = n.conn.Drain()
	n.conn.Close()
}
