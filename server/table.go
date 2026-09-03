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

type createTableRequest struct {
	Name string `json:"name"`
}

// tableWriteFailure resolves a write's nil/false failure result into the right status: 403 if
// the table exists under a different owner, 404 if it doesn't exist at all.
func tableWriteFailure(c *gin.Context) {
	tbl, err := data.GetTable(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	if tbl != nil {
		c.JSON(http.StatusForbidden, apiv.ErrorVO{Error: "forbidden", Message: "Caller does not own this resource"})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{})
}

func setupTableHandlers(g *gin.Engine, store persistence.CacheStore, ttls cachettl.Config, authzClient *authz.Client) {
	ttl := ttls.TTL("tables")
	viewer := authz.ResolveViewer(authzClient)
	owner := authz.RequireOwner()

	g.GET("/users/:user_id/tables", viewer, cache.CachePage(store, ttl, listTables))
	g.GET("/users/:user_id/tables/:id", viewer, cache.CachePage(store, ttl, getTable))

	g.POST("/users/:user_id/tables", viewer, owner, createTable)
	g.PUT("/users/:user_id/tables/:id", viewer, owner, updateTableName)
	g.DELETE("/users/:user_id/tables/:id", viewer, owner, deleteTable)
	g.POST("/users/:user_id/tables/:id/volumes", viewer, owner, addTableVolume)
	g.DELETE("/users/:user_id/tables/:id/volumes/:volume_id", viewer, owner, removeTableVolume)
	g.PUT("/users/:user_id/tables/:id/visibility", viewer, owner, setTableVisibility)
}

// List a user's tables.
//
//	@Summary		List tables
//	@Description	List a user's tables, filtered to what the caller may see.
//	@Tags			tables
//	@Produce		json
//	@Param			user_id	path		string	true	"User ID"
//	@Success		200		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/tables [get]
func listTables(c *gin.Context) {
	tables, err := data.ListTablesByUser(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	viewer := authz.Viewer(c)
	vos := make([]interface{}, 0, len(tables))
	for _, t := range tables {
		if v := data.TableToVO(t, viewer, false, false); v != nil {
			vos = append(vos, v)
		}
	}
	c.JSON(http.StatusOK, vos)
}

// Get one table.
//
//	@Summary		Get table
//	@Tags			tables
//	@Produce		json
//	@Param			user_id	path		string	true	"User ID"
//	@Param			id		path		string	true	"Table ID"
//	@Success		200		{object}	interface{}
//	@Failure		404		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/tables/{id} [get]
func getTable(c *gin.Context) {
	tbl, err := data.GetTable(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	if tbl == nil {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	vo := data.TableToVO(tbl, authz.Viewer(c), false, false)
	if vo == nil {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	c.JSON(http.StatusOK, vo)
}

// Create a table.
//
//	@Summary		Create table
//	@Description	Create a new table, defaulting to private visibility.
//	@Tags			tables
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"
//	@Param			body	body		createTableRequest	true	"Table name"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/tables [post]
func createTable(c *gin.Context) {
	var req createTableRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "name is required"})
		return
	}
	tbl, err := data.CreateTable(c.Request.Context(), c.Param("user_id"), req.Name, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.TableToVO(tbl, authz.Viewer(c), false, false))
}

// Rename a table.
//
//	@Summary		Rename table
//	@Tags			tables
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"
//	@Param			id		path		string				true	"Table ID"
//	@Param			body	body		createTableRequest	true	"New name"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/tables/{id} [put]
func updateTableName(c *gin.Context) {
	var req createTableRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "name is required"})
		return
	}
	tbl, err := data.UpdateTableName(c.Request.Context(), c.Param("id"), c.Param("user_id"), req.Name, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if tbl == nil {
		tableWriteFailure(c)
		return
	}
	c.JSON(http.StatusOK, data.TableToVO(tbl, authz.Viewer(c), false, false))
}

// Delete a table.
//
//	@Summary		Delete table
//	@Tags			tables
//	@Param			user_id	path	string	true	"User ID"
//	@Param			id		path	string	true	"Table ID"
//	@Success		204
//	@Failure		500	{object}	interface{}
//	@Router			/users/{user_id}/tables/{id} [delete]
func deleteTable(c *gin.Context) {
	deleted, err := data.DeleteTable(c.Request.Context(), c.Param("id"), c.Param("user_id"), authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if !deleted {
		tableWriteFailure(c)
		return
	}
	c.Status(http.StatusNoContent)
}

// Add a volume to a table.
//
//	@Summary		Add table volume
//	@Tags			tables
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"
//	@Param			id		path		string				true	"Table ID"
//	@Param			body	body		volumeEntryRequest	true	"Volume to add"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/tables/{id}/volumes [post]
func addTableVolume(c *gin.Context) {
	var req volumeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VolumeID == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "volume_id is required"})
		return
	}
	tbl, err := data.AddTableVolume(c.Request.Context(), c.Param("id"), c.Param("user_id"), req.VolumeID, req.VolumeTitle, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if tbl == nil {
		tableWriteFailure(c)
		return
	}
	c.JSON(http.StatusOK, data.TableToVO(tbl, authz.Viewer(c), false, false))
}

// Remove a volume from a table.
//
//	@Summary		Remove table volume
//	@Tags			tables
//	@Produce		json
//	@Param			user_id		path		string	true	"User ID"
//	@Param			id			path		string	true	"Table ID"
//	@Param			volume_id	path		string	true	"Volume ID"
//	@Success		200			{object}	interface{}
//	@Failure		500			{object}	interface{}
//	@Router			/users/{user_id}/tables/{id}/volumes/{volume_id} [delete]
func removeTableVolume(c *gin.Context) {
	tbl, err := data.RemoveTableVolume(c.Request.Context(), c.Param("id"), c.Param("user_id"), c.Param("volume_id"), authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if tbl == nil {
		tableWriteFailure(c)
		return
	}
	c.JSON(http.StatusOK, data.TableToVO(tbl, authz.Viewer(c), false, false))
}

// Set a table's visibility.
//
//	@Summary		Set table visibility
//	@Tags			tables
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID"
//	@Param			id		path		string				true	"Table ID"
//	@Param			body	body		visibilityRequest	true	"New visibility"
//	@Success		200		{object}	interface{}
//	@Failure		400		{object}	interface{}
//	@Failure		500		{object}	interface{}
//	@Router			/users/{user_id}/tables/{id}/visibility [put]
func setTableVisibility(c *gin.Context) {
	var req visibilityRequest
	v, ok := bindVisibility(c, &req)
	if !ok {
		return
	}
	tbl, err := data.SetTableVisibility(c.Request.Context(), c.Param("id"), c.Param("user_id"), v, authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if tbl == nil {
		tableWriteFailure(c)
		return
	}
	c.JSON(http.StatusOK, data.TableToVO(tbl, authz.Viewer(c), false, false))
}
