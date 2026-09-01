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
// referencing library entry, then drop those users' cached library pages.
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

	affected, err := data.UpdateLibraryEntryTitleByVolume(ctx, volumeID, title)
	if err != nil {
		return fmt.Errorf("update library titles for volume %s: %w", volumeID, err)
	}

	h.invalidateLibraryCache(affected)
	logging.Logger.Info("volume title synced",
		"event_id", event.EventID, "volume_id", volumeID, "users", len(affected))
	return nil
}

// invalidateLibraryCache drops the cached library-list page for each user whose entries changed,
// so their next read reflects the new title. gin-contrib/cache keys the library GET by its URL.
func (h *VolumeEventHandler) invalidateLibraryCache(userIDs []string) {
	if h.cache == nil {
		return
	}
	for _, uid := range userIDs {
		key := cache.CreateKey("/users/" + uid + "/library")
		if err := h.cache.Delete(key); err != nil && !errors.Is(err, persistence.ErrCacheMiss) {
			logging.Logger.Warn("library cache invalidation failed", "user_id", uid, "error", err)
		}
	}
}
