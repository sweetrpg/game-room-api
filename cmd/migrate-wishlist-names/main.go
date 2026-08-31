// migrate-wishlist-names runs the one-time migration that backfills the default
// wishlist name onto documents stored before the multi-wishlist cutover.
//
// Safe to run more than once: data.MigrateWishlistNames leaves already-named
// wishlists untouched and reports the number of documents updated. Wired into
// .github/workflows/prepare-release.yaml as a release-blocking step so the old
// singular routes can be retired only after every existing wishlist carries a
// name.
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

// migrateTimeout bounds a single run against a slow or unreachable database so a
// hang cannot stall the release pipeline forever.
const migrateTimeout = 60 * time.Second

func main() {
	_ = godotenv.Load(".env")

	logging.Init()

	database.SetupDatabase()
	defer database.TeardownDatabase()

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()

	count, err := data.MigrateWishlistNames(ctx)
	if err != nil {
		logging.Logger.Error("Wishlist name migration failed", "error", err.Error())
		os.Exit(1)
	}

	logging.Logger.Info("Wishlist name migration complete", "updated", count)
}
