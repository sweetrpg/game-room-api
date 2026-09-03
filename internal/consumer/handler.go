package consumer

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/sweetrpg/common.go/logging"
	"github.com/sweetrpg/game-room-data.go/data"
)

// VolumeEventHandler applies volume.updated events: refresh the denormalized title on every
// referencing library entry, table volume, and wishlist entry, then drop the affected users'
// cached list and detail pages.
type VolumeEventHandler struct {
	cache persistence.CacheStore
}

func NewVolumeEventHandler(store persistence.CacheStore) *VolumeEventHandler {
	return &VolumeEventHandler{cache: store}
}

func (h *VolumeEventHandler) HandleVolumeUpdated(ctx context.Context, event *EventEnvelope) error {
	volumeID := event.EntityID
	if volumeID == "" {
		return fmt.Errorf("volume.updated event %s has no entity_id", event.EventID)
	}
	title, _ := event.Data["title"].(string)
	if title == "" {
		logging.Logger.Warn("volume.updated event has no data.title",
			"event_id", event.EventID, "volume_id", volumeID)
		return nil
	}

	// All three refreshes run for every event. A failure in any of them returns an error so the
	// event is Nak'ed and redelivered; each refresh is idempotent, so the retry is safe.
	libUsers, err := data.UpdateLibraryEntryTitleByVolume(ctx, volumeID, title)
	if err != nil {
		return fmt.Errorf("update library titles for volume %s: %w", volumeID, err)
	}
	tableUsers, err := data.UpdateTableVolumeTitleByVolume(ctx, volumeID, title)
	if err != nil {
		return fmt.Errorf("update table titles for volume %s: %w", volumeID, err)
	}
	wishlistUsers, err := data.UpdateWishlistEntryTitleByVolume(ctx, volumeID, title)
	if err != nil {
		return fmt.Errorf("update wishlist titles for volume %s: %w", volumeID, err)
	}

	affected := union(libUsers, tableUsers, wishlistUsers)
	h.invalidate(ctx, affected, volumeID)
	logging.Logger.Info("volume title synced",
		"event_id", event.EventID, "volume_id", volumeID, "users", len(affected))
	return nil
}

// invalidate drops every affected user's cached library, table, and wishlist pages - the list
// pages unconditionally (their URLs derive from the user ID alone) and the detail pages for
// whichever of their tables/wishlists actually reference the changed volume. gin-contrib/cache
// keys a GET by its URL.
func (h *VolumeEventHandler) invalidate(ctx context.Context, userIDs []string, volumeID string) {
	if h.cache == nil {
		return
	}
	for _, uid := range userIDs {
		base := "/users/" + uid
		h.drop(uid, cache.CreateKey(base+"/library"))
		h.drop(uid, cache.CreateKey(base+"/tables"))
		h.drop(uid, cache.CreateKey(base+"/wishlists"))

		if tables, err := data.ListTablesByUser(ctx, uid); err != nil {
			logging.Logger.Warn("table detail cache invalidation lookup failed", "user_id", uid, "error", err)
		} else {
			for _, t := range tables {
				if contains(t.VolumeIDs, volumeID) {
					h.drop(uid, cache.CreateKey(base+"/tables/"+t.ID))
				}
			}
		}

		if wls, err := data.ListWishlistsByUser(ctx, uid); err != nil {
			logging.Logger.Warn("wishlist detail cache invalidation lookup failed", "user_id", uid, "error", err)
		} else {
			for _, wl := range wls {
				for _, e := range wl.Entries {
					if e.VolumeID == volumeID {
						h.drop(uid, cache.CreateKey(base+"/wishlists/"+wl.ID))
						break
					}
				}
			}
		}
	}
}

func (h *VolumeEventHandler) drop(userID, key string) {
	if err := h.cache.Delete(key); err != nil && !errors.Is(err, persistence.ErrCacheMiss) {
		logging.Logger.Warn("cache invalidation failed", "user_id", userID, "key", key, "error", err)
	}
}

func union(lists ...[]string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, l := range lists {
		for _, s := range l {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
