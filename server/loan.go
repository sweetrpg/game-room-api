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
)

// createLoanRequest accepts exactly one of BorrowerUserID or BorrowerName; when BorrowerUserID is
// given, BorrowerName must also be supplied (the caller already has the linked user's display
// name from its own user search - see design.md's borrower resolution decision).
type createLoanRequest struct {
	VolumeID       string  `json:"volume_id" example:"vol-123"`
	BorrowerUserID *string `json:"borrower_user_id" example:"user-456"`
	BorrowerName   string  `json:"borrower_name" example:"Jordan"`
}

// loanWriteFailure resolves a write's nil/false failure result into the right status: 403 if the
// loan exists under a different lender, 404 if it doesn't exist at all.
func loanWriteFailure(c *gin.Context) {
	loan, err := data.GetLoan(c.Request.Context(), c.Param("loan_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	if loan != nil {
		c.JSON(http.StatusForbidden, apiv.ErrorVO{Error: "forbidden", Message: "Caller does not own this resource"})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{})
}

func setupLoanHandlers(g *gin.Engine, store persistence.CacheStore, ttls cachettl.Config, authzClient *authz.Client) {
	ttl := ttls.TTL("loans")
	viewer := authz.ResolveViewer(authzClient, "game-room-api")
	owner := authz.RequireOwner()

	g.GET("/users/:user_id/loans", viewer, owner, cache.CachePage(store, ttl, listLoansLent))
	g.GET("/users/:user_id/loans/borrowed", viewer, owner, cache.CachePage(store, ttl, listLoansBorrowed))
	g.POST("/users/:user_id/loans", viewer, owner, createLoan)
	g.POST("/users/:user_id/loans/:loan_id/return", viewer, owner, returnLoan)
	g.DELETE("/users/:user_id/loans/:loan_id", viewer, owner, deleteLoan)
}

// List loans lent out by a user.
//
//	@Summary		List lent loans
//	@Description	List the volumes a user has lent out, both open and returned.
//	@Tags			loans
//	@Produce		json
//	@Param			user_id	path		string	true	"User ID"	example(user-123)
//	@Success		200		{array}		LoanVO
//	@Failure		500		{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/loans [get]
func listLoansLent(c *gin.Context) {
	loans, err := data.ListLoansLentBy(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	vos := make([]interface{}, 0, len(loans))
	for _, l := range loans {
		vos = append(vos, data.LoanToVO(l))
	}
	c.JSON(http.StatusOK, vos)
}

// List loans borrowed by a user.
//
//	@Summary		List borrowed loans
//	@Description	List the volumes a platform-linked user has borrowed, both open and returned.
//	@Tags			loans
//	@Produce		json
//	@Param			user_id	path		string	true	"User ID"	example(user-456)
//	@Success		200		{array}		LoanVO
//	@Failure		500		{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/loans/borrowed [get]
func listLoansBorrowed(c *gin.Context) {
	loans, err := data.ListLoansBorrowedBy(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	vos := make([]interface{}, 0, len(loans))
	for _, l := range loans {
		vos = append(vos, data.LoanToVO(l))
	}
	c.JSON(http.StatusOK, vos)
}

// Create a loan.
//
//	@Summary		Create loan
//	@Description	Lend a catalog volume to a platform-linked user or a free-form name. Exactly one of borrower_user_id or borrower_name is required; when borrower_user_id is given, borrower_name must also be supplied as that user's display name.
//	@Tags			loans
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		string				true	"User ID (the lender)"	example(user-123)
//	@Param			body	body		createLoanRequest	true	"Loan details"
//	@Success		200		{object}	LoanVO
//	@Failure		400		{object}	apiv.ErrorVO	"volume_id missing, or borrower fields don't satisfy the exactly-one-of rule"
//	@Failure		409		{object}	apiv.ErrorVO	"volume is already out on an open loan"
//	@Failure		500		{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/loans [post]
func createLoan(c *gin.Context) {
	var req createLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VolumeID == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "volume_id is required"})
		return
	}
	if (req.BorrowerUserID == nil || *req.BorrowerUserID == "") == (req.BorrowerName == "") {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "exactly one of borrower_user_id or borrower_name is required"})
		return
	}
	if req.BorrowerUserID != nil && *req.BorrowerUserID != "" && req.BorrowerName == "" {
		c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: "borrower_name is required alongside borrower_user_id"})
		return
	}

	loan, err := data.CreateLoan(c.Request.Context(), c.Param("user_id"), req.VolumeID, req.BorrowerUserID, req.BorrowerName, authz.Viewer(c))
	if err != nil {
		if errors.Is(err, data.ErrLoanBorrowerRequired) {
			c.JSON(http.StatusBadRequest, apiv.ErrorVO{Error: "bad_request", Message: err.Error()})
			return
		}
		if errors.Is(err, data.ErrLoanAlreadyOpen) {
			c.JSON(http.StatusConflict, apiv.ErrorVO{Error: "conflict", Message: err.Error()})
			return
		}
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.LoanToVO(loan))
}

// Mark a loan returned.
//
//	@Summary		Return loan
//	@Description	Mark a lent-out loan as returned. Lender-only.
//	@Tags			loans
//	@Produce		json
//	@Param			user_id		path		string	true	"User ID (the lender)"	example(user-123)
//	@Param			loan_id		path		string	true	"Loan ID"				example(loan-789)
//	@Success		200			{object}	LoanVO
//	@Failure		403			{object}	apiv.ErrorVO		"loan exists but is owned by a different lender"
//	@Failure		404			{object}	map[string]interface{}	"no loan with this ID exists"
//	@Failure		500			{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/loans/{loan_id}/return [post]
func returnLoan(c *gin.Context) {
	loan, err := data.MarkLoanReturned(c.Request.Context(), c.Param("loan_id"), c.Param("user_id"), authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if loan == nil {
		loanWriteFailure(c)
		return
	}
	c.JSON(http.StatusOK, data.LoanToVO(loan))
}

// Delete a loan.
//
//	@Summary		Delete loan
//	@Description	Permanently remove a loan record. Lender-only.
//	@Tags			loans
//	@Param			user_id		path	string	true	"User ID (the lender)"	example(user-123)
//	@Param			loan_id		path	string	true	"Loan ID"				example(loan-789)
//	@Success		204
//	@Failure		403	{object}	apiv.ErrorVO		"loan exists but is owned by a different lender"
//	@Failure		404	{object}	map[string]interface{}	"no loan with this ID exists"
//	@Failure		500	{object}	apiv.ErrorVO
//	@Router			/users/{user_id}/loans/{loan_id} [delete]
func deleteLoan(c *gin.Context) {
	deleted, err := data.DeleteLoan(c.Request.Context(), c.Param("loan_id"), c.Param("user_id"), authz.Viewer(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if !deleted {
		loanWriteFailure(c)
		return
	}
	c.Status(http.StatusNoContent)
}
