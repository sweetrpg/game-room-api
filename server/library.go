package server

import (
	"net/http"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	apiv "github.com/sweetrpg/api-core.go/vo"
	"github.com/sweetrpg/shelf-api/authz"
	"github.com/sweetrpg/shelf-api/cachettl"
	"github.com/sweetrpg/shelf-data.go/data"
	"github.com/sweetrpg/shelf-objects.go/models"
)

func setupLibraryHandlers(g *gin.Engine, store persistence.CacheStore, ttls cachettl.Config, authzClient *authz.Client) {
	ttl := ttls.TTL("library")
	viewer := authz.ResolveViewer(authzClient)
	owner := authz.RequireOwner()

	g.GET("/users/:user_id/library", viewer, cache.CachePage(store, ttl, getLibrary))

	g.POST("/users/:user_id/library/entries", viewer, owner, addLibraryEntry)
	g.DELETE("/users/:user_id/library/entries/:volume_id", viewer, owner, removeLibraryEntry)
	g.PUT("/users/:user_id/library/entries/:volume_id/visibility", viewer, owner, setLibraryEntryVisibility)
	g.PUT("/users/:user_id/library/default-visibility", viewer, owner, setLibraryDefaultVisibility)
	g.POST("/users/:user_id/library/default-visibility/preview", viewer, owner, previewLibraryDefaultVisibility)
}

// Get a user's library.
//
//	@Summary		Get library
//	@Description	Get a user's library, filtered to what the caller may see.
//	@Tags			library
//	@Produce		json
//	@Param			user_id	path		string	true	"User ID"
//	@Success		200		{object}	interface{}
//	@Failure		500		{object}	interface{}
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
//	@Param			user_id	path		string				true	"User ID"
//	@Param			body	body		volumeEntryRequest	true	"Volume to add"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/library/entries [post]
func addLibraryEntry(c *gin.Context) {
	var req volumeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VolumeID == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "volume_id is required"})
		return
	}
	lib, err := data.AddLibraryEntry(c.Request.Context(), c.Param("user_id"), req.VolumeID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Remove a library entry.
//
//	@Summary		Remove library entry
//	@Tags			library
//	@Produce		json
//	@Param			user_id		path		string	true	"User ID"
//	@Param			volume_id	path		string	true	"Volume ID"
//	@Success		200			{object}	interface{}
//	@Failure		500			{object}	interface{}
//	@Router			/users/{user_id}/library/entries/{volume_id} [delete]
func removeLibraryEntry(c *gin.Context) {
	lib, err := data.RemoveLibraryEntry(c.Request.Context(), c.Param("user_id"), c.Param("volume_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Set a library entry's visibility override.
//
//	@Summary		Set library entry visibility override
//	@Description	Set (or clear, with an empty visibility) a per-entry visibility override.
//	@Tags			library
//	@Accept			json
//	@Produce		json
//	@Param			user_id		path		string				true	"User ID"
//	@Param			volume_id	path		string				true	"Volume ID"
//	@Param			body		body		visibilityRequest	true	"New override (empty clears it)"
//	@Success		200			{object}	interface{}
//	@Failure		400			{object}	interface{}
//	@Failure		500			{object}	interface{}
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

	lib, err := data.SetLibraryEntryVisibilityOverride(c.Request.Context(), c.Param("user_id"), c.Param("volume_id"), override)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LibraryToVO(lib, authz.Viewer(c), false, false))
}

// Set the library's default visibility.
//
//	@Summary		Set library default visibility
//	@Tags			library
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"
//	@Param			body	body		visibilityRequest	true	"New default visibility"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/library/default-visibility [put]
func setLibraryDefaultVisibility(c *gin.Context) {
	var req visibilityRequest
	v, ok := bindVisibility(c, &req)
	if !ok {
		return
	}
	lib, err := data.SetLibraryDefaultVisibility(c.Request.Context(), c.Param("user_id"), v)
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
//	@Param			user_id	path		string				true	"User ID"
//	@Param			body	body		visibilityRequest	true	"Proposed new default visibility"
//	@Success		200		{object}	previewResponse
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
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
