package consumer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/sweetrpg/game-room-data.go/data"
	"github.com/sweetrpg/mongodb.go/constants"
	"github.com/sweetrpg/mongodb.go/database"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	uri := os.Getenv("TEST_DB_URI")
	if uri == "" {
		t.Skip("TEST_DB_URI not set - this repo's CI has no Mongo service container yet, skip DB-backed tests")
	}
	_ = os.Setenv(constants.DB_URI, uri)
	database.SetupDatabase()
}

func event(volumeID, title string) *EventEnvelope {
	return &EventEnvelope{
		EventID:    "evt-" + primitive.NewObjectID().Hex(),
		EntityType: "volume",
		EntityID:   volumeID,
		Action:     "updated",
		Data:       map[string]interface{}{"title": title},
	}
}

// seedKey primes a cache entry so its later absence proves the handler dropped it.
func seedKey(store persistence.CacheStore, url string) string {
	key := cache.CreateKey(url)
	_ = store.Set(key, "cached", time.Minute)
	return key
}

func missing(t *testing.T, store persistence.CacheStore, key string) {
	t.Helper()
	var v string
	if err := store.Get(key, &v); err != persistence.ErrCacheMiss {
		t.Errorf("key %q still cached (err=%v)", key, err)
	}
}

func present(t *testing.T, store persistence.CacheStore, key string) {
	t.Helper()
	var v string
	if err := store.Get(key, &v); err != nil {
		t.Errorf("key %q unexpectedly gone (err=%v)", key, err)
	}
}

func TestHandlerSyncsAllThreeAndInvalidatesPages(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()
	uid := primitive.NewObjectID().Hex()
	volumeID := primitive.NewObjectID().Hex()

	if _, err := data.AddLibraryEntry(ctx, uid, volumeID, "Old", uid); err != nil {
		t.Fatalf("seed library: %v", err)
	}
	tbl, err := data.CreateTable(ctx, uid, "T", uid)
	if err != nil {
		t.Fatalf("seed table: %v", err)
	}
	if _, err := data.AddTableVolume(ctx, tbl.ID, uid, volumeID, "Old", uid); err != nil {
		t.Fatalf("seed table volume: %v", err)
	}
	wl, err := data.CreateWishlist(ctx, uid, "W", uid)
	if err != nil {
		t.Fatalf("seed wishlist: %v", err)
	}
	if _, err := data.AddWishlistEntry(ctx, wl.ID, uid, volumeID, "Old", uid); err != nil {
		t.Fatalf("seed wishlist entry: %v", err)
	}

	store := persistence.NewInMemoryStore(time.Minute)
	base := "/users/" + uid
	libList := seedKey(store, base+"/library")
	tblList := seedKey(store, base+"/tables")
	tblDetail := seedKey(store, base+"/tables/"+tbl.ID)
	wlList := seedKey(store, base+"/wishlists")
	wlDetail := seedKey(store, base+"/wishlists/"+wl.ID)
	unrelated := seedKey(store, "/users/"+primitive.NewObjectID().Hex()+"/tables")

	h := NewVolumeEventHandler(store)
	if err := h.HandleVolumeUpdated(ctx, event(volumeID, "New Title")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	lib, _ := data.GetLibraryByUser(ctx, uid)
	if lib.Entries[0].VolumeTitle != "New Title" {
		t.Errorf("library title = %q", lib.Entries[0].VolumeTitle)
	}
	gotTbl, _ := data.GetTable(ctx, tbl.ID)
	if gotTbl.VolumeTitles[volumeID] != "New Title" {
		t.Errorf("table title = %q", gotTbl.VolumeTitles[volumeID])
	}
	gotWl, _ := data.GetWishlist(ctx, wl.ID)
	if gotWl.Entries[0].VolumeTitle != "New Title" {
		t.Errorf("wishlist title = %q", gotWl.Entries[0].VolumeTitle)
	}

	for _, k := range []string{libList, tblList, tblDetail, wlList, wlDetail} {
		missing(t, store, k)
	}
	present(t, store, unrelated)
}

func TestHandlerNoReferencesStillAcksAndInvalidatesNothing(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	store := persistence.NewInMemoryStore(time.Minute)
	survivor := seedKey(store, "/users/"+primitive.NewObjectID().Hex()+"/tables")

	h := NewVolumeEventHandler(store)
	if err := h.HandleVolumeUpdated(ctx, event(primitive.NewObjectID().Hex(), "Nobody Has This")); err != nil {
		t.Fatalf("handle: %v", err)
	}
	present(t, store, survivor)
}

func TestHandlerMissingEntityIDErrors(t *testing.T) {
	h := NewVolumeEventHandler(nil)
	if err := h.HandleVolumeUpdated(context.Background(), &EventEnvelope{EventID: "evt-x"}); err == nil {
		t.Fatal("expected error for event with no entity_id")
	}
}
