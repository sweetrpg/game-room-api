package server

import (
	"errors"
	"net/http"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	apiv "github.com/sweetrpg/api-core.go/vo"
	"github.com/sweetrpg/authz-client.go/authz"
	"github.com/sweetrpg/game-room-api/cachettl"
	"github.com/sweetrpg/game-room-data.go/data"
	"github.com/sweetrpg/game-room-objects.go/models"
)

func setupLibraryHandlers(g *gin.Engine, store persistence.CacheStore, ttls cachettl.Config, authzClient *authz.Client) {
	ttl := ttls.TTL("library")
	viewer := authz.ResolveViewer(authzClient, "game-room-api")
	owner := authz.RequireOwner()

	g.GET("/users/:user_id/library", viewer, cache.CachePage(store, ttl, getLibrary))

	g.POST("/users/:user_id/library/entries", viewer, owner, addLibraryEntry)
	g.DELETE("/users/:user_id/library/entries/:volume_id", viewer, owner, removeLibraryEntry)
	g.PUT("/users/:user_id/library/entries/:volume_id/visibility", viewer, owner, setLibraryEntryVisibility)
	g.PUT("/users/:user_id/library/entries/:volume_id/title", viewer, owner, updateLibraryEntryTitle)
	g.PUT("/users/:user_id/library/default-visibility", viewer, owner, setLibraryDefaultVisibility)
	g.POST("/users/:user_id/library/default-visibility/preview", viewer, owner, previewLibraryDefaultVisibility)
}

// Get a user's library.
//
//	@Summary		Get library
//	@Description	Get a user's library, filtered to what the caller may see.
//	@Tags			library
//	@Produce		json
//	@Param			user_id	path		string	true	"User ID"	example(user-123)
//	@Success		200		{object}	LibraryVO
//	@Failure		500		{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/library [get]
func getLibrary(c *gin.Context) {
	userID := c.Param("user_id")
	lib, err := data.GetLibraryByUser(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err)
		return
	}
	if lib == nil {
		lib = &models.Library{UserID: userID, DefaultVisibility: models.VisibilityPrivate}
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Add a library entry.
//
//	@Summary		Add library entry
//	@Description	Link a catalog volume into the caller's own library.
//	@Tags			library
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"	example(user-123)
//	@Param			body	body		volumeEntryRequest	true	"Volume to add"
//	@Success		200		{object}	LibraryVO
//	@Failure		400		{object}	apiv.ErrorVO	"volume_id missing"
//	@Failure		500		{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/library/entries [post]
func addLibraryEntry(c *gin.Context) {
	var req volumeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VolumeID == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "volume_id is required"})
		return
	}
	lib, err := data.AddLibraryEntry(c.Request.Context(), c.Param("user_id"), req.VolumeID, req.VolumeTitle, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Remove a library entry.
//
//	@Summary		Remove library entry
//	@Description	Unlink a catalog volume from the caller's own library.
//	@Tags			library
//	@Produce		json
//	@Param			user_id		path		string	true	"User ID"	example(user-123)
//	@Param			volume_id	path		string	true	"Volume ID"	example(vol-123)
//	@Success		200			{object}	LibraryVO
//	@Failure		500			{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/library/entries/{volume_id} [delete]
func removeLibraryEntry(c *gin.Context) {
	lib, err := data.RemoveLibraryEntry(c.Request.Context(), c.Param("user_id"), c.Param("volume_id"), authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Set a library entry's visibility override.
//
//	@Summary		Set library entry visibility override
//	@Description	Set (or clear, with an empty visibility) a per-entry visibility override. Valid visibility values: public, friends, friends_of_friends, private.
//	@Tags			library
//	@Accept			json
//	@Produce		json
//	@Param			user_id		path		string				true	"User ID"	example(user-123)
//	@Param			volume_id	path		string				true	"Volume ID"	example(vol-123)
//	@Param			body		body		visibilityRequest	true	"New override (empty clears it)"
//	@Success		200			{object}	LibraryVO
//	@Failure		400			{object}	apiv.ErrorVO	"invalid body or invalid visibility value"
//	@Failure		500			{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/library/entries/{volume_id}/visibility [put]
func setLibraryEntryVisibility(c *gin.Context) {
	var req visibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "invalid body"})
		return
	}

	var override *models.Visibility
	if req.Visibility != "" {
		v, ok := parseVisibility(req.Visibility)
		if !ok {
			c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "invalid visibility"})
			return
		}
		override = &v
	}

	lib, err := data.SetLibraryEntryVisibilityOverride(c.Request.Context(), c.Param("user_id"), c.Param("volume_id"), override, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Update a library entry's display title.
//
//	@Summary		Update library entry title
//	@Description	Refresh the denormalized title snapshot on a single library entry.
//	@Tags			library
//	@Accept			json
//	@Produce		json
//	@Param			user_id		path		string			true	"User ID"	example(user-123)
//	@Param			volume_id	path		string			true	"Volume ID"	example(vol-123)
//	@Param			body		body		titleRequest	true	"New title"
//	@Success		200			{object}	LibraryVO
//	@Failure		400			{object}	apiv.ErrorVO		"title missing"
//	@Failure		404			{object}	map[string]interface{}	"no entry for this volume_id in the library"
//	@Failure		500			{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/library/entries/{volume_id}/title [put]
func updateLibraryEntryTitle(c *gin.Context) {
	var req titleRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "title is required"})
		return
	}
	lib, err := data.UpdateLibraryEntryTitle(c.Request.Context(), c.Param("user_id"), c.Param("volume_id"), req.Title, authz.Viewer(c))
	if err != nil {
		if errors.Is(err, data.ErrLibraryEntryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{})
			return
		}
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Set the library's default visibility.
//
//	@Summary		Set library default visibility
//	@Description	Set the visibility applied to library entries that have no per-entry override. Valid visibility values: public, friends, friends_of_friends, private.
//	@Tags			library
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"	example(user-123)
//	@Param			body	body		visibilityRequest	true	"New default visibility"
//	@Success		200		{object}	LibraryVO
//	@Failure		400		{object}	apiv.ErrorVO	"invalid body or invalid visibility value"
//	@Failure		500		{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/library/default-visibility [put]
func setLibraryDefaultVisibility(c *gin.Context) {
	var req visibilityRequest
	v, ok := bindVisibility(c, &req)
	if !ok {
		return
	}
	lib, err := data.SetLibraryDefaultVisibility(c.Request.Context(), c.Param("user_id"), v, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Preview a library default-visibility change.
//
//	@Summary		Preview library default-visibility change
//	@Description	Dry-run a default-visibility change, returning the volume IDs of entries that would become more exposed - backs the warning dialog before the caller commits to the change.
//	@Tags			library
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"	example(user-123)
//	@Param			body	body		visibilityRequest	true	"Proposed new default visibility"
//	@Success		200		{object}	previewResponse
//	@Failure		400		{object}	apiv.ErrorVO	"invalid body or invalid visibility value"
//	@Failure		500		{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/library/default-visibility/preview [post]
func previewLibraryDefaultVisibility(c *gin.Context) {
	var req visibilityRequest
	v, ok := bindVisibility(c, &req)
	if !ok {
		return
	}

	lib, err := data.GetLibraryByUser(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	if lib == nil {
		c.JSON(http.StatusOK, previewResponse{AffectedVolumeIDs: []string{}})
		return
	}

	c.JSON(http.StatusOK, previewResponse{AffectedVolumeIDs: data.PreviewLibraryDefaultChange(lib, v)})
}
