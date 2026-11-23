package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sweetrpg/authz-client.go/authz"
	"github.com/sweetrpg/game-room-data.go/data"
)

// VolumeSummaryVO is one user's Game Room state for one catalog volume.
type VolumeSummaryVO struct {
	Library  SummaryLibraryVO   `json:"library"`
	Wishlist *SummaryWishlistVO `json:"wishlist"`
	Tables   []SummaryTableVO   `json:"tables"`
	Loan     *SummaryLoanVO     `json:"loan"`
}

// SummaryLibraryVO reports library membership; AddedAt is null when the volume is not present.
type SummaryLibraryVO struct {
	Present bool       `json:"present" example:"true"`
	AddedAt *time.Time `json:"added_at"`
}

// SummaryWishlistVO names the one wishlist holding the volume.
type SummaryWishlistVO struct {
	ID   string `json:"id" example:"wl-789"`
	Name string `json:"name" example:"Birthday wishlist"`
}

// SummaryTableVO names a table that includes the volume.
type SummaryTableVO struct {
	ID   string `json:"id" example:"tbl-456"`
	Name string `json:"name" example:"Friday night"`
}

// SummaryLoanVO describes the open loan of the volume, if any.
type SummaryLoanVO struct {
	ID             string    `json:"id" example:"loan-123"`
	BorrowerUserID *string   `json:"borrower_user_id"`
	BorrowerName   string    `json:"borrower_name" example:"Jordan"`
	LentAt         time.Time `json:"lent_at"`
}

func setupSummaryHandlers(g *gin.Engine, authzClient *authz.Client) {
	viewer := authz.ResolveViewer(authzClient, "game-room-api")
	// Not cached: the Catalog tab must reflect an add or remove made a moment ago.
	g.GET("/users/:user_id/volumes/:volume_id/summary", viewer, authz.RequireOwner(), getVolumeSummary)
}

// Get a volume's Game Room summary.
//
//	@Summary		Volume summary
//	@Description	The caller's library, wishlist, table and open-loan state for one catalog volume. Owner-only.
//	@Tags			summary
//	@Produce		json
//	@Param			user_id		path		string	true	"User ID"	example(user-123)
//	@Param			volume_id	path		string	true	"Volume ID"	example(vol-123)
//	@Success		200			{object}	VolumeSummaryVO
//	@Failure		403			{object}	map[string]interface{}	"path user is not the caller"
//	@Failure		500			{object}	map[string]interface{}
//	@Router			/users/{user_id}/volumes/{volume_id}/summary [get]
func getVolumeSummary(c *gin.Context) {
	ctx, userID, volumeID := c.Request.Context(), c.Param("user_id"), c.Param("volume_id")
	out := VolumeSummaryVO{Tables: []SummaryTableVO{}}

	entry, err := data.GetLibraryEntry(ctx, userID, volumeID)
	if err != nil {
		internalError(c, err)
		return
	}
	if entry != nil {
		out.Library = SummaryLibraryVO{Present: true, AddedAt: &entry.AddedAt}
	}

	wl, err := data.FindWishlistByVolume(ctx, userID, volumeID)
	if err != nil {
		internalError(c, err)
		return
	}
	if wl != nil {
		out.Wishlist = &SummaryWishlistVO{ID: wl.ID, Name: wl.Name}
	}

	tables, err := data.ListTablesByVolume(ctx, userID, volumeID)
	if err != nil {
		internalError(c, err)
		return
	}
	for _, t := range tables {
		out.Tables = append(out.Tables, SummaryTableVO{ID: t.ID, Name: t.Name})
	}

	loan, err := data.FindOpenLoanByVolume(ctx, userID, volumeID)
	if err != nil {
		internalError(c, err)
		return
	}
	if loan != nil {
		out.Loan = &SummaryLoanVO{ID: loan.ID, BorrowerUserID: loan.BorrowerUserID, BorrowerName: loan.BorrowerName, LentAt: loan.LentAt}
	}

	c.JSON(http.StatusOK, out)
}
