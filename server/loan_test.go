package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	objvo "github.com/sweetrpg/game-room-objects.go/vo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestListLoansRequestsForLenderAndBorrower(t *testing.T) {
	setupTestDB(t)
	lender := primitive.NewObjectID().Hex()
	borrower := primitive.NewObjectID().Hex()
	volumeID := primitive.NewObjectID().Hex()

	createC, createW := newTestContext(t, http.MethodPost, "/users/"+lender+"/loans",
		createLoanRequest{VolumeID: volumeID, BorrowerUserID: &borrower, BorrowerName: "Borrower"},
		gin.Params{{Key: "user_id", Value: lender}}, lender)
	createLoan(createC)
	if createW.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", createW.Code, createW.Body.String())
	}

	lentC, lentW := newTestContext(t, http.MethodGet, "/users/"+lender+"/loans", nil, gin.Params{{Key: "user_id", Value: lender}}, lender)
	listLoansLent(lentC)
	var lent []objvo.LoanVO
	if err := json.Unmarshal(lentW.Body.Bytes(), &lent); err != nil {
		t.Fatalf("decode lent list: %v", err)
	}
	if len(lent) != 1 {
		t.Fatalf("len(lent) = %d, want 1", len(lent))
	}

	borrowedC, borrowedW := newTestContext(t, http.MethodGet, "/users/"+borrower+"/loans/borrowed", nil, gin.Params{{Key: "user_id", Value: borrower}}, borrower)
	listLoansBorrowed(borrowedC)
	var borrowed []objvo.LoanVO
	if err := json.Unmarshal(borrowedW.Body.Bytes(), &borrowed); err != nil {
		t.Fatalf("decode borrowed list: %v", err)
	}
	if len(borrowed) != 1 {
		t.Fatalf("len(borrowed) = %d, want 1", len(borrowed))
	}
}

func TestCreateLoanRequestBothBorrowerFields(t *testing.T) {
	setupTestDB(t)
	lender := primitive.NewObjectID().Hex()
	volumeID := primitive.NewObjectID().Hex()

	linkedC, linkedW := newTestContext(t, http.MethodPost, "/users/"+lender+"/loans",
		createLoanRequest{VolumeID: volumeID, BorrowerUserID: strPtr(primitive.NewObjectID().Hex()), BorrowerName: "Linked Friend"},
		gin.Params{{Key: "user_id", Value: lender}}, lender)
	createLoan(linkedC)
	if linkedW.Code != http.StatusOK {
		t.Fatalf("linked create status = %d, body = %s", linkedW.Code, linkedW.Body.String())
	}

	freeFormC, freeFormW := newTestContext(t, http.MethodPost, "/users/"+lender+"/loans",
		createLoanRequest{VolumeID: primitive.NewObjectID().Hex(), BorrowerName: "Real Life Friend"},
		gin.Params{{Key: "user_id", Value: lender}}, lender)
	createLoan(freeFormC)
	if freeFormW.Code != http.StatusOK {
		t.Fatalf("free-form create status = %d, body = %s", freeFormW.Code, freeFormW.Body.String())
	}
}

func TestCreateLoanRequestRejectsNeitherBorrowerField(t *testing.T) {
	setupTestDB(t)
	lender := primitive.NewObjectID().Hex()

	c, w := newTestContext(t, http.MethodPost, "/users/"+lender+"/loans",
		createLoanRequest{VolumeID: primitive.NewObjectID().Hex()},
		gin.Params{{Key: "user_id", Value: lender}}, lender)
	createLoan(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body.String())
	}
}

func TestLoanNonOwnerRequestsAreForbidden(t *testing.T) {
	setupTestDB(t)
	owner := primitive.NewObjectID().Hex()
	intruder := primitive.NewObjectID().Hex()

	createC, createW := newTestContext(t, http.MethodPost, "/users/"+owner+"/loans",
		createLoanRequest{VolumeID: primitive.NewObjectID().Hex(), BorrowerName: "Friend"},
		gin.Params{{Key: "user_id", Value: owner}}, owner)
	createLoan(createC)
	var created objvo.LoanVO
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	returnC, _ := newTestContext(t, http.MethodPost, "/users/"+intruder+"/loans/"+created.ID+"/return", nil,
		gin.Params{{Key: "user_id", Value: intruder}, {Key: "loan_id", Value: created.ID}}, intruder)
	returnLoan(returnC)
	if returnC.Writer.Status() != http.StatusForbidden {
		t.Fatalf("return by non-owner status = %d, want 403", returnC.Writer.Status())
	}

	deleteC, _ := newTestContext(t, http.MethodDelete, "/users/"+intruder+"/loans/"+created.ID, nil,
		gin.Params{{Key: "user_id", Value: intruder}, {Key: "loan_id", Value: created.ID}}, intruder)
	deleteLoan(deleteC)
	if deleteC.Writer.Status() != http.StatusForbidden {
		t.Fatalf("delete by non-owner status = %d, want 403", deleteC.Writer.Status())
	}

	missingC, _ := newTestContext(t, http.MethodPost, "/users/"+owner+"/loans/"+primitive.NewObjectID().Hex()+"/return", nil,
		gin.Params{{Key: "user_id", Value: owner}, {Key: "loan_id", Value: primitive.NewObjectID().Hex()}}, owner)
	returnLoan(missingC)
	if missingC.Writer.Status() != http.StatusNotFound {
		t.Fatalf("return of nonexistent loan status = %d, want 404", missingC.Writer.Status())
	}
}

func strPtr(s string) *string { return &s }
