package server

import (
	"net/http"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	apiv "github.com/sweetrpg/api-core.go/vo"
	"github.com/sweetrpg/game-room-api/authz"
	"github.com/sweetrpg/game-room-api/cachettl"
	"github.com/sweetrpg/game-room-data.go/data"
	"github.com/sweetrpg/game-room-objects.go/models"
)

func setupWishlistHandlers(g *gin.Engine, store persistence.CacheStore, ttls cachettl.Config, authzClient *authz.Client) {
	ttl := ttls.TTL("wishlist")
	viewer := authz.ResolveViewer(authzClient)
	owner := authz.RequireOwner()

	g.GET("/users/:user_id/wishlist", viewer, cache.CachePage(store, ttl, getWishlist))

	g.POST("/users/:user_id/wishlist/entries", viewer, owner, addWishlistEntry)
	g.DELETE("/users/:user_id/wishlist/entries/:volume_id", viewer, owner, removeWishlistEntry)
	g.PUT("/users/:user_id/wishlist/visibility", viewer, owner, setWishlistVisibility)
}

// Get a user's wishlist.
//
//	@Summary		Get wishlist
//	@Description	Get a user's wishlist, or 404 if the caller may not see it.
//	@Tags			wishlist
//	@Produce		json
//	@Param			user_id	path		string	true	"User ID"
//	@Success		200		{object}	interface{}
//	@Failure		404		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/wishlist [get]
func getWishlist(c *gin.Context) {
	userID := c.Param("user_id")
	wl, err := data.GetWishlistByUser(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err)
		return
	}
	if wl == nil {
		wl = &models.Wishlist{UserID: userID, Visibility: models.VisibilityPrivate}
	}
	vo := data.WishlistToVO(wl, authz.Viewer(c), false, false)
	if vo == nil {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	c.JSON(http.StatusOK, vo)
}

// Add a wishlist entry.
//
//	@Summary		Add wishlist entry
//	@Tags			wishlist
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"
//	@Param			body	body		volumeEntryRequest	true	"Volume to add"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/wishlist/entries [post]
func addWishlistEntry(c *gin.Context) {
	var req volumeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VolumeID == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "volume_id is required"})
		return
	}
	wl, err := data.AddWishlistEntry(c.Request.Context(), c.Param("user_id"), req.VolumeID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.WishlistToVO(wl, authz.Viewer(c), false, false))
}

// Remove a wishlist entry.
//
//	@Summary		Remove wishlist entry
//	@Tags			wishlist
//	@Produce		json
//	@Param			user_id		path		string	true	"User ID"
//	@Param			volume_id	path		string	true	"Volume ID"
//	@Success		200			{object}	interface{}
//	@Failure		500			{object}	interface{}
//	@Router			/users/{user_id}/wishlist/entries/{volume_id} [delete]
func removeWishlistEntry(c *gin.Context) {
	wl, err := data.RemoveWishlistEntry(c.Request.Context(), c.Param("user_id"), c.Param("volume_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.WishlistToVO(wl, authz.Viewer(c), false, false))
}

// Set the wishlist's visibility.
//
//	@Summary		Set wishlist visibility
//	@Tags			wishlist
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"
//	@Param			body	body		visibilityRequest	true	"New visibility"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/wishlist/visibility [put]
func setWishlistVisibility(c *gin.Context) {
	var req visibilityRequest
	v, ok := bindVisibility(c, &req)
	if !ok {
		return
	}
	wl, err := data.SetWishlistVisibility(c.Request.Context(), c.Param("user_id"), v)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.WishlistToVO(wl, authz.Viewer(c), false, false))
}
