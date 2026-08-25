package server

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	apiv "github.com/sweetrpg/api-core.go/vo"
	"github.com/sweetrpg/game-room-objects.go/models"
)

type volumeEntryRequest struct {
	VolumeID string `json:"volume_id"`
}

type visibilityRequest struct {
	Visibility string `json:"visibility"`
}

type previewResponse struct {
	AffectedVolumeIDs []string `json:"affected_volume_ids"`
}

var validVisibilities = map[string]models.Visibility{
	string(models.VisibilityPublic):           models.VisibilityPublic,
	string(models.VisibilityFriends):          models.VisibilityFriends,
	string(models.VisibilityFriendsOfFriends): models.VisibilityFriendsOfFriends,
	string(models.VisibilityPrivate):          models.VisibilityPrivate,
}

func parseVisibility(s string) (models.Visibility, bool) {
	v, ok := validVisibilities[s]
	return v, ok
}

// bindVisibility decodes a visibilityRequest body and validates its Visibility field, writing a
// 400 response and returning ok=false on any failure - callers can return immediately.
func bindVisibility(c *gin.Context, req *visibilityRequest) (models.Visibility, bool) {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "invalid body"})
		return "", false
	}
	v, ok := parseVisibility(req.Visibility)
	if !ok {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "invalid visibility"})
		return "", false
	}
	return v, true
}

func internalError(c *gin.Context, err error) {
	sentry.CaptureException(err)
	c.JSON(http.StatusInternalServerError, apiv.ErrorVO{Error: "internal_error", Message: err.Error()})
}
