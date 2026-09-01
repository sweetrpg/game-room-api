package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sweetrpg/common.go/logging"
	"github.com/sweetrpg/common.go/util"
)

// EventEnvelope is the platform event contract (see sweetrpg/platform docs/messaging-nats.md).
// Unknown fields are ignored so the envelope can grow without breaking this consumer.
type EventEnvelope struct {
	EventID    string                 `json:"event_id"`
	OccurredAt string                 `json:"occurred_at"`
	Source     string                 `json:"source"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Action     string                 `json:"action"`
	Revision   int                    `json:"revision"`
	Data       map[string]interface{} `json:"data"`
}

type EventHandler interface {
	HandleVolumeUpdated(ctx context.Context, event *EventEnvelope) error
}

const (
	defaultStreamName      = "CATALOG_EVENTS"
	defaultDurableConsumer = "game-room-api-volume-title-sync"
	filterSubject          = "catalog.events.volume.updated"
)

// Consumer binds a durable JetStream pull consumer on the catalog events stream and applies
// volume.updated events to library entries. It runs as a background worker of the API process
// and is also startable standalone via cmd/consumer.
type Consumer struct {
	handler EventHandler
	conn    *nats.Conn
	cc      jetstream.ConsumeContext
}

func New(handler EventHandler) *Consumer {
	return &Consumer{handler: handler}
}

// Start connects to NATS, binds the durable consumer, and begins consuming in the background.
// Safe to pair with Stop even if Start returns an error.
func (c *Consumer) Start(ctx context.Context) error {
	natsURL := util.GetEnv("NATS_URL", "nats://localhost:4222")
	streamName := util.GetEnv("NATS_STREAM", defaultStreamName)
	durableName := util.GetEnv("NATS_DURABLE_NAME", defaultDurableConsumer)

	opts := []nats.Option{nats.Name(durableName)}
	if creds := os.Getenv("NATS_CREDS"); creds != "" {
		opts = append(opts, nats.UserCredentials(creds))
	} else if user := os.Getenv("NATS_USER"); user != "" {
		opts = append(opts, nats.UserInfo(user, os.Getenv("NATS_PASSWORD")))
	}
	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return fmt.Errorf("connect to NATS at %s: %w", natsURL, err)
	}
	c.conn = conn

	js, err := jetstream.New(conn)
	if err != nil {
		return fmt.Errorf("create JetStream context: %w", err)
	}

	// Idempotent: matches the declaratively-managed Consumer CRD where a controller creates it,
	// and creates it directly for local runs where none does. The durable name is stable so a
	// redeploy resumes from the last acknowledged position.
	cons, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:       durableName,
		FilterSubject: filterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("bind durable consumer %q on stream %q: %w", durableName, streamName, err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) { c.process(ctx, msg) })
	if err != nil {
		return fmt.Errorf("start consuming: %w", err)
	}
	c.cc = cc

	logging.Logger.Info("volume-title-sync consumer bound",
		"stream", streamName, "durable", durableName, "subject", filterSubject)
	return nil
}

// Stop halts consumption and closes the connection. Idempotent.
func (c *Consumer) Stop(context.Context) error {
	if c.cc != nil {
		c.cc.Stop()
		c.cc = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	logging.Logger.Info("volume-title-sync consumer stopped")
	return nil
}

func (c *Consumer) process(ctx context.Context, msg jetstream.Msg) {
	var env EventEnvelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		// A poison message would redeliver forever; ack it and move on.
		logging.Logger.Error("dropping unparseable event", "error", err)
		_ = msg.Ack()
		return
	}

	if env.EntityType != "volume" || env.Action != "updated" {
		// The subject filter should exclude these; ack defensively if the server widens it.
		_ = msg.Ack()
		return
	}

	if err := c.handler.HandleVolumeUpdated(ctx, &env); err != nil {
		// Do not ack: JetStream redelivers after the ack wait.
		logging.Logger.Error("volume.updated handling failed, will retry",
			"event_id", env.EventID, "volume_id", env.EntityID, "error", err)
		_ = msg.Nak()
		return
	}
	if err := msg.Ack(); err != nil {
		logging.Logger.Error("ack failed", "event_id", env.EventID, "volume_id", env.EntityID, "error", err)
	}
}
