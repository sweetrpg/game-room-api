package consumer

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sweetrpg/common.go/logging"
)

func TestMain(m *testing.M) {
	logging.Init()
	os.Exit(m.Run())
}

// startTestJetStream spins up an in-process NATS server with JetStream and the CATALOG_EVENTS
// stream, and returns a JetStream handle plus the client URL.
func startTestJetStream(t *testing.T) (jetstream.JetStream, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := server.NewServer(&server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		StoreDir:  dir,
	})
	if err != nil {
		t.Fatalf("new test NATS server: %v", err)
	}
	go s.Start()
	if !s.ReadyForConnections(5 * time.Second) {
		s.Shutdown()
		t.Fatal("test NATS server not ready")
	}
	t.Cleanup(s.Shutdown)

	conn, err := nats.Connect(s.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(conn.Close)

	js, err := jetstream.New(conn)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	if _, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "CATALOG_EVENTS",
		Subjects: []string{"catalog.events.>"},
	}); err != nil {
		t.Fatalf("create stream: %v", err)
	}
	return js, s.ClientURL()
}

func startConsumer(t *testing.T, url, durable string, h EventHandler) *Consumer {
	t.Helper()
	t.Setenv("NATS_URL", url)
	t.Setenv("NATS_STREAM", "CATALOG_EVENTS")
	t.Setenv("NATS_DURABLE_NAME", durable)
	c := New(h)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("consumer start: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background()) })
	return c
}

func publish(t *testing.T, js jetstream.JetStream, subject string, env EventEnvelope) {
	t.Helper()
	payload, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := js.Publish(context.Background(), subject, payload); err != nil {
		t.Fatalf("publish %s: %v", subject, err)
	}
}

func volumeUpdated(id, volID, title string) EventEnvelope {
	return EventEnvelope{
		EventID:    id,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		Source:     "catalog-api",
		EntityType: "volume",
		EntityID:   volID,
		Action:     "updated",
		Revision:   1,
		Data:       map[string]interface{}{"title": title},
	}
}

func TestConsumerStartBindsDurable(t *testing.T) {
	_, url := startTestJetStream(t)
	c := startConsumer(t, url, "test-bind", &mockHandler{})
	if c.cc == nil {
		t.Fatal("consume context not established")
	}
}

func TestConsumerProcessesVolumeUpdated(t *testing.T) {
	js, url := startTestJetStream(t)
	h := &mockHandler{}
	startConsumer(t, url, "test-process", h)

	publish(t, js, "catalog.events.volume.updated", volumeUpdated("evt-1", "vol-1", "New Title"))

	waitFor(t, func() bool { return h.count() == 1 })
	if got := h.last(); got != "evt-1" {
		t.Fatalf("last event id = %q, want evt-1", got)
	}
}

func TestConsumerFiltersNonVolumeUpdated(t *testing.T) {
	js, url := startTestJetStream(t)
	h := &mockHandler{}
	startConsumer(t, url, "test-filter", h)

	publish(t, js, "catalog.events.volume.created", volumeUpdated("evt-c", "vol-1", ""))
	publish(t, js, "catalog.events.volume.deleted", volumeUpdated("evt-d", "vol-1", ""))
	publish(t, js, "catalog.events.person.updated", EventEnvelope{EventID: "evt-p", EntityType: "person", Action: "updated"})
	// A volume.updated to prove the consumer is live and only this one lands.
	publish(t, js, "catalog.events.volume.updated", volumeUpdated("evt-u", "vol-1", "T"))

	waitFor(t, func() bool { return h.count() == 1 })
	time.Sleep(300 * time.Millisecond) // give any stray delivery a chance to (wrongly) arrive
	if h.count() != 1 || h.last() != "evt-u" {
		t.Fatalf("count=%d last=%q, want 1 / evt-u", h.count(), h.last())
	}
}

func TestConsumerRedeliversOnHandlerFailure(t *testing.T) {
	js, url := startTestJetStream(t)
	h := &mockHandler{failTimes: 2}
	startConsumer(t, url, "test-retry", h)

	publish(t, js, "catalog.events.volume.updated", volumeUpdated("evt-retry", "vol-1", "Eventually"))

	// Fails twice (Nak -> prompt redelivery), succeeds on the third delivery.
	waitFor(t, func() bool { return h.count() == 1 })
	if h.fails() != 2 {
		t.Fatalf("failures = %d, want 2", h.fails())
	}
}

func TestConsumerHandlesRedeliveredEvent(t *testing.T) {
	js, url := startTestJetStream(t)
	h := &mockHandler{}
	startConsumer(t, url, "test-idem", h)

	env := volumeUpdated("evt-dup", "vol-1", "Same")
	publish(t, js, "catalog.events.volume.updated", env)
	publish(t, js, "catalog.events.volume.updated", env)

	waitFor(t, func() bool { return h.count() == 2 })
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met within 5s")
}

type mockHandler struct {
	mu        sync.Mutex
	handled   int
	failed    int
	failTimes int
	lastID    string
}

func (h *mockHandler) HandleVolumeUpdated(_ context.Context, event *EventEnvelope) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failed < h.failTimes {
		h.failed++
		return errSimulated
	}
	h.handled++
	h.lastID = event.EventID
	return nil
}

func (h *mockHandler) count() int   { h.mu.Lock(); defer h.mu.Unlock(); return h.handled }
func (h *mockHandler) fails() int   { h.mu.Lock(); defer h.mu.Unlock(); return h.failed }
func (h *mockHandler) last() string { h.mu.Lock(); defer h.mu.Unlock(); return h.lastID }

var errSimulated = &simulatedError{}

type simulatedError struct{}

func (*simulatedError) Error() string { return "simulated handler failure" }
