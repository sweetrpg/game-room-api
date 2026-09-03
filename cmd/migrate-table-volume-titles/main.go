// migrate-table-volume-titles runs the one-time structural migration that ensures every table
// document has a volume_titles object, setting it to an empty object on tables stored before
// the denormalized-title sidecar existed.
//
// Safe to run more than once: data.MigrateTableVolumeTitles leaves tables that already have the
// field untouched and reports the number of documents updated. It populates no titles - those
// fill in as volumes are re-added or as catalog volume.updated events arrive. Run it against
// each environment's database after deploying the release that adds the sidecar.
package main

import (
	"context"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/sweetrpg/common.go/logging"
	"github.com/sweetrpg/game-room-data.go/data"
	"github.com/sweetrpg/mongodb.go/database"
)

// migrateTimeout bounds a single run against a slow or unreachable database so a hang cannot
// stall the release pipeline forever.
const migrateTimeout = 60 * time.Second

func main() {
	_ = godotenv.Load(".env")

	logging.Init()

	database.SetupDatabase()
	defer database.TeardownDatabase()

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()

	count, err := data.MigrateTableVolumeTitles(ctx)
	if err != nil {
		logging.Logger.Error("Table volume-title migration failed", "error", err.Error())
		os.Exit(1)
	}

	logging.Logger.Info("Table volume-title migration complete", "updated", count)
}
