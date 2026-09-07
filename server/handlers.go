package server

import (
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	"github.com/sweetrpg/authz-client.go/authz"
	"github.com/sweetrpg/game-room-api/cachettl"
)

func SetupHandlers(g *gin.Engine, cache persistence.CacheStore, ttls cachettl.Config, authzClient *authz.Client) {
	setupLibraryHandlers(g, cache, ttls, authzClient)
	setupWishlistHandlers(g, cache, ttls, authzClient)
	setupTableHandlers(g, cache, ttls, authzClient)
	setupStatusHandlers(g)
}
