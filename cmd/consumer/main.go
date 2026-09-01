package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sweetrpg/common.go/logging"
	"github.com/sweetrpg/game-room-api/internal/consumer"
	"github.com/sweetrpg/mongodb.go/database"
)

func main() {
	_ = godotenv.Load(".env")

	logging.Init()

	// Setup database connection
	database.SetupDatabase()
	defer database.TeardownDatabase()

	// Standalone runs (local dev) have no cache to invalidate; the in-process worker in
	// cmd/game-room-api passes the API's real store. A nil store is handled as a no-op.
	handler := consumer.NewVolumeEventHandler(nil)

	// Create and start the consumer
	c := consumer.New(handler)
	if err := c.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop consumer: %v", err)
	}
}
