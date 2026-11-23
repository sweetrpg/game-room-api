package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func summaryParams(userID, volumeID string) gin.Params {
	return gin.Params{{Key: "user_id", Value: userID}, {Key: "volume_id", Value: volumeID}}
}

func getSummary(t *testing.T, userID, volumeID string) VolumeSummaryVO {
	t.Helper()
	c, w := newTestContext(t, http.MethodGet, "/users/"+userID+"/volumes/"+volumeID+"/summary", nil, summaryParams(userID, volumeID), userID)
	getVolumeSummary(c)
	if w.Code != http.StatusOK {
		t.Fatalf("summary status = %d, body = %s", w.Code, w.Body.String())
	}
	var got VolumeSummaryVO
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	return got
}

func TestVolumeSummaryEmptyForUnknownVolume(t *testing.T) {
	setupTestDB(t)
	userID := primitive.NewObjectID().Hex()

	got := getSummary(t, userID, "vol-none")
	if got.Library.Present || got.Wishlist != nil || got.Loan != nil || len(got.Tables) != 0 {
		t.Fatalf("summary = %+v, want empty", got)
	}
}

func TestVolumeSummaryReportsLibraryWishlistTableAndLoan(t *testing.T) {
	setupTestDB(t)
	userID := primitive.NewObjectID().Hex()
	volumeID := "vol-1"

	c, w := newTestContext(t, http.MethodPost, "/users/"+userID+"/library/entries", volumeEntryRequest{VolumeID: volumeID, VolumeTitle: "Vol"}, gin.Params{{Key: "user_id", Value: userID}}, userID)
	addLibraryEntry(c)
	if w.Code != http.StatusOK {
		t.Fatalf("add library status = %d, body = %s", w.Code, w.Body.String())
	}
	wl := createTestWishlist(t, userID, "Birthday")
	addTestWishlistEntry(t, userID, wl.ID, volumeID, http.StatusOK)
	lc, lw := newTestContext(t, http.MethodPost, "/users/"+userID+"/loans", createLoanRequest{VolumeID: volumeID, BorrowerName: "Jordan"}, gin.Params{{Key: "user_id", Value: userID}}, userID)
	createLoan(lc)
	if lw.Code != http.StatusOK {
		t.Fatalf("create loan status = %d, body = %s", lw.Code, lw.Body.String())
	}

	got := getSummary(t, userID, volumeID)
	if !got.Library.Present || got.Library.AddedAt == nil {
		t.Errorf("library = %+v, want present with added_at", got.Library)
	}
	if got.Wishlist == nil || got.Wishlist.ID != wl.ID || got.Wishlist.Name != "Birthday" {
		t.Errorf("wishlist = %+v, want Birthday", got.Wishlist)
	}
	if got.Loan == nil || got.Loan.BorrowerName != "Jordan" {
		t.Errorf("loan = %+v, want borrower Jordan", got.Loan)
	}
}

func TestVolumeSummaryIsScopedToUser(t *testing.T) {
	setupTestDB(t)
	owner := primitive.NewObjectID().Hex()
	other := primitive.NewObjectID().Hex()
	wl := createTestWishlist(t, owner, "Mine")
	addTestWishlistEntry(t, owner, wl.ID, "vol-1", http.StatusOK)

	got := getSummary(t, other, "vol-1")
	if got.Wishlist != nil {
		t.Fatalf("other user sees wishlist %+v, want nil", got.Wishlist)
	}
}

func TestAddWishlistEntryConflictNamesExistingWishlist(t *testing.T) {
	setupTestDB(t)
	userID := primitive.NewObjectID().Hex()
	first := createTestWishlist(t, userID, "First")
	second := createTestWishlist(t, userID, "Second")
	addTestWishlistEntry(t, userID, first.ID, "vol-1", http.StatusOK)

	w := addTestWishlistEntry(t, userID, second.ID, "vol-1", http.StatusConflict)

	var body wishlistConflictVO
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode conflict: %v", err)
	}
	if body.WishlistID != first.ID || body.WishlistName != "First" {
		t.Fatalf("conflict = %+v, want First (%s)", body, first.ID)
	}
}
