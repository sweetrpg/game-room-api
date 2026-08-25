package authz

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apiv "github.com/sweetrpg/api-core.go/vo"
	"github.com/sweetrpg/common.go/logging"
)

const viewerContextKey = "authz.viewer"

// ResolveViewer returns Gin middleware that resolves the caller's identity from a bearer token,
// when present, and stashes it in the Gin context (read via Viewer(c)) for handlers to use in
// visibility filtering and ownership checks. A missing token, or one auth-api rejects, resolves
// to an anonymous viewer ("") rather than aborting the request - reads must stay accessible to
// anonymous callers for anything public.
func ResolveViewer(client *Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token == "" {
			c.Set(viewerContextKey, "")
			c.Next()
			return
		}

		result, err := client.Check(c.Request.Context(), token, "game-room-api")
		if err != nil {
			logging.Logger.Debug("authz check failed, treating caller as anonymous", "error", err.Error())
			c.Set(viewerContextKey, "")
			c.Next()
			return
		}

		c.Set(viewerContextKey, result.Sub)
		c.Next()
	}
}

// Viewer returns the caller's resolved subject, or "" for an anonymous viewer.
func Viewer(c *gin.Context) string {
	if v, ok := c.Get(viewerContextKey); ok {
		if sub, ok := v.(string); ok {
			return sub
		}
	}
	return ""
}

// RequireOwner aborts with 403 unless the resolved viewer matches the :user_id path param -
// every Game Room write endpoint is scoped to the caller's own collection.
func RequireOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		viewer := Viewer(c)
		if viewer == "" || viewer != c.Param("user_id") {
			c.AbortWithStatusJSON(http.StatusForbidden, apiv.ErrorVO{
				Error:   "forbidden",
				Message: "Caller does not own this resource",
			})
			return
		}
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	const prefix = "Bearer "
	auth := c.GetHeader("Authorization")
	if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
		return ""
	}
	return auth[len(prefix):]
}
