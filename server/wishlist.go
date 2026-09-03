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
)

type createWishlistRequest struct {
	Name string `json:"name"`
}

// wishlistWriteFailure resolves a write's nil/false failure result into the right status: 403
// if the wishlist exists under a different owner, 404 if it doesn't exist at all.
func wishlistWriteFailure(c *gin.Context) {
	wl, err := data.GetWishlist(c.Request.Context(), c.Param("wishlist_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	if wl != nil {
		c.JSON(http.StatusForbidden, apiv.ErrorVO{Error: "forbidden", Message: "Caller does not own this resource"})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{})
}

func setupWishlistHandlers(g *gin.Engine, store persistence.CacheStore, ttls cachettl.Config, authzClient *authz.Client) {
	ttl := ttls.TTL("wishlist")
	viewer := authz.ResolveViewer(authzClient)
	owner := authz.RequireOwner()

	g.GET("/users/:user_id/wishlists", viewer, cache.CachePage(store, ttl, listWishlists))
	g.GET("/users/:user_id/wishlists/:wishlist_id", viewer, cache.CachePage(store, ttl, getWishlistByID))
	g.POST("/users/:user_id/wishlists", viewer, owner, createWishlist)
	g.DELETE("/users/:user_id/wishlists/:wishlist_id", viewer, owner, deleteWishlist)

	g.POST("/users/:user_id/wishlists/:wishlist_id/entries", viewer, owner, addWishlistEntry)
	g.DELETE("/users/:user_id/wishlists/:wishlist_id/entries/:volume_id", viewer, owner, removeWishlistEntry)
	g.PUT("/users/:user_id/wishlists/:wishlist_id/visibility", viewer, owner, setWishlistVisibility)
}

// List a user's wishlists.
//
//	@Summary		List wishlists
//	@Description	List a user's wishlists, filtered to what the caller may see.
//	@Tags			wishlist
//	@Produce		json
//	@Param			user_id	path		string	true	"User ID"
//	@Success		200		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/wishlists [get]
func listWishlists(c *gin.Context) {
	wls, err := data.ListWishlistsByUser(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	viewer := authz.Viewer(c)
	vos := make([]interface{}, 0, len(wls))
	for _, wl := range wls {
		if v := data.WishlistToVO(wl, viewer, false, false); v != nil {
			vos = append(vos, v)
		}
	}
	c.JSON(http.StatusOK, vos)
}

// Get one wishlist.
//
//	@Summary		Get wishlist
//	@Tags			wishlist
//	@Produce		json
//	@Param			user_id			path		string	true	"User ID"
//	@Param			wishlist_id		path		string	true	"Wishlist ID"
//	@Success		200				{object}	interface{}
//	@Failure		404				{object}	interface{}
//	@Failure		500				{object}	interface{}
//	@Router			/users/{user_id}/wishlists/{wishlist_id} [get]
func getWishlistByID(c *gin.Context) {
	wl, err := data.GetWishlist(c.Request.Context(), c.Param("wishlist_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	if wl == nil {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	vo := data.WishlistToVO(wl, authz.Viewer(c), false, false)
	if vo == nil {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	c.JSON(http.StatusOK, vo)
}

// Create a wishlist.
//
//	@Summary		Create wishlist
//	@Description	Create a new named wishlist, defaulting to private visibility.
//	@Tags			wishlist
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string					true	"User ID"
//	@Param			body	body		createWishlistRequest	true	"Wishlist name"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/wishlists [post]
func createWishlist(c *gin.Context) {
	var req createWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "name is required"})
		return
	}
	wl, err := data.CreateWishlist(c.Request.Context(), c.Param("user_id"), req.Name, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.WishlistToVO(wl, authz.Viewer(c), false, false))
}

// Delete a wishlist.
//
//	@Summary		Delete wishlist
//	@Tags			wishlist
//	@Param			user_id			path	string	true	"User ID"
//	@Param			wishlist_id		path	string	true	"Wishlist ID"
//	@Success		204
//	@Failure		500	{object}	interface{}
//	@Router			/users/{user_id}/wishlists/{wishlist_id} [delete]
func deleteWishlist(c *gin.Context) {
	deleted, err := data.DeleteWishlist(c.Request.Context(), c.Param("wishlist_id"), c.Param("user_id"), authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if !deleted {
		wishlistWriteFailure(c)
		return
	}
	c.Status(http.StatusNoContent)
}

// Add a wishlist entry.
//
//	@Summary		Add wishlist entry
//	@Tags			wishlist
//	@Accept			json
//	@Produce		json
//	@Param			user_id			path		string				true	"User ID"
//	@Param			wishlist_id		path		string				true	"Wishlist ID"
//	@Param			body			body		volumeEntryRequest	true	"Volume to add"
//	@Success		200				{object}	interface{}
//	@Failure		400				{object}	interface{}
//	@Failure		500				{object}	interface{}
//	@Router			/users/{user_id}/wishlists/{wishlist_id}/entries [post]
func addWishlistEntry(c *gin.Context) {
	var req volumeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VolumeID == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "volume_id is required"})
		return
	}
	wl, err := data.AddWishlistEntry(c.Request.Context(), c.Param("wishlist_id"), c.Param("user_id"), req.VolumeID, req.VolumeTitle, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if wl == nil {
		wishlistWriteFailure(c)
		return
	}
	c.JSON(http.StatusOK, data.WishlistToVO(wl, authz.Viewer(c), false, false))
}

// Remove a wishlist entry.
//
//	@Summary		Remove wishlist entry
//	@Tags			wishlist
//	@Produce		json
//	@Param			user_id			path		string	true	"User ID"
//	@Param			wishlist_id		path		string	true	"Wishlist ID"
//	@Param			volume_id		path		string	true	"Volume ID"
//	@Success		200				{object}	interface{}
//	@Failure		500				{object}	interface{}
//	@Router			/users/{user_id}/wishlists/{wishlist_id}/entries/{volume_id} [delete]
func removeWishlistEntry(c *gin.Context) {
	wl, err := data.RemoveWishlistEntry(c.Request.Context(), c.Param("wishlist_id"), c.Param("user_id"), c.Param("volume_id"), authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if wl == nil {
		wishlistWriteFailure(c)
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
//	@Param			user_id			path		string				true	"User ID"
//	@Param			wishlist_id		path		string				true	"Wishlist ID"
//	@Param			body			body		visibilityRequest	true	"New visibility"
//	@Success		200				{object}	interface{}
//	@Failure		400				{object}	interface{}
//	@Failure		500				{object}	interface{}
//	@Router			/users/{user_id}/wishlists/{wishlist_id}/visibility [put]
func setWishlistVisibility(c *gin.Context) {
	var req visibilityRequest
	v, ok := bindVisibility(c, &req)
	if !ok {
		return
	}
	wl, err := data.SetWishlistVisibility(c.Request.Context(), c.Param("wishlist_id"), c.Param("user_id"), v, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if wl == nil {
		wishlistWriteFailure(c)
		return
	}
	c.JSON(http.StatusOK, data.WishlistToVO(wl, authz.Viewer(c), false, false))
}
