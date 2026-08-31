package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sweetrpg/common.go/logging"
	objvo "github.com/sweetrpg/game-room-objects.go/vo"
	"github.com/sweetrpg/mongodb.go/constants"
	"github.com/sweetrpg/mongodb.go/database"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	uri := os.Getenv("TEST_DB_URI")
	if uri == "" {
		t.Skip("TEST_DB_URI not set - this repo's CI has no Mongo service container yet, skip DB-backed tests")
	}
	_ = os.Setenv(constants.DB_URI, uri)
	logging.Init()
	database.SetupDatabase()
}

func newTestContext(t *testing.T, method, path string, body any, params gin.Params, viewerID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	buf := &bytes.Buffer{}
	if body != nil {
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = params
	c.Set("authz.viewer", viewerID)
	return c, w
}

func TestListWishlistsRequest(t *testing.T) {
	setupTestDB(t)
	userID := primitive.NewObjectID().Hex()

	createC, createW := newTestContext(t, http.MethodPost, "/users/"+userID+"/wishlists", createWishlistRequest{Name: "Birthday"}, gin.Params{{Key: "user_id", Value: userID}}, userID)
	createWishlist(createC)
	if createW.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", createW.Code, createW.Body.String())
	}

	createC2, createW2 := newTestContext(t, http.MethodPost, "/users/"+userID+"/wishlists", createWishlistRequest{Name: "Con haul"}, gin.Params{{Key: "user_id", Value: userID}}, userID)
	createWishlist(createC2)
	if createW2.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", createW2.Code, createW2.Body.String())
	}

	listC, listW := newTestContext(t, http.MethodGet, "/users/"+userID+"/wishlists", nil, gin.Params{{Key: "user_id", Value: userID}}, userID)
	listWishlists(listC)
	if listW.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listW.Code, listW.Body.String())
	}

	var got []objvo.WishlistVO
	if err := json.Unmarshal(listW.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(wishlists) = %d, want 2", len(got))
	}
}

func TestDeleteWishlistRequestRemovesOnlyThatWishlist(t *testing.T) {
	setupTestDB(t)
	userID := primitive.NewObjectID().Hex()

	createC, createW := newTestContext(t, http.MethodPost, "/users/"+userID+"/wishlists", createWishlistRequest{Name: "Remove me"}, gin.Params{{Key: "user_id", Value: userID}}, userID)
	createWishlist(createC)
	var created objvo.WishlistVO
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	deleteC, _ := newTestContext(t, http.MethodDelete, "/users/"+userID+"/wishlists/"+created.ID, nil,
		gin.Params{{Key: "user_id", Value: userID}, {Key: "wishlist_id", Value: created.ID}}, userID)
	deleteWishlist(deleteC)
	// c.Status() alone only buffers the header on gin's writer; it's flushed to the recorder on
	// the next Write() call, which a no-body 204 response never makes outside gin's own request
	// lifecycle. Assert against the writer's buffered status instead of the recorder's.
	if deleteC.Writer.Status() != http.StatusNoContent {
		t.Fatalf("delete status = %d", deleteC.Writer.Status())
	}

	getC, getW := newTestContext(t, http.MethodGet, "/users/"+userID+"/wishlists/"+created.ID, nil,
		gin.Params{{Key: "user_id", Value: userID}, {Key: "wishlist_id", Value: created.ID}}, userID)
	getWishlistByID(getC)
	if getW.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", getW.Code)
	}
}

func TestNonOwnerWriteRequestsAreForbidden(t *testing.T) {
	setupTestDB(t)
	owner := primitive.NewObjectID().Hex()
	intruder := primitive.NewObjectID().Hex()

	createC, createW := newTestContext(t, http.MethodPost, "/users/"+owner+"/wishlists", createWishlistRequest{Name: "Mine"}, gin.Params{{Key: "user_id", Value: owner}}, owner)
	createWishlist(createC)
	var created objvo.WishlistVO
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	// The intruder's request passes owner middleware's :user_id check by naming itself as the
	// path owner, but targets someone else's wishlist_id - the IDOR this whole path is guarding
	// against. RequireOwner is bypassed in these tests (handlers are called directly), so this
	// exercises exactly the data-layer ownership check that must catch it instead.
	deleteC, _ := newTestContext(t, http.MethodDelete, "/users/"+intruder+"/wishlists/"+created.ID, nil,
		gin.Params{{Key: "user_id", Value: intruder}, {Key: "wishlist_id", Value: created.ID}}, intruder)
	deleteWishlist(deleteC)
	if deleteC.Writer.Status() != http.StatusForbidden {
		t.Fatalf("delete by non-owner status = %d, want 403", deleteC.Writer.Status())
	}

	entryC, entryW := newTestContext(t, http.MethodPost, "/users/"+intruder+"/wishlists/"+created.ID+"/entries", volumeEntryRequest{VolumeID: "vol-1"},
		gin.Params{{Key: "user_id", Value: intruder}, {Key: "wishlist_id", Value: created.ID}}, intruder)
	addWishlistEntry(entryC)
	if entryW.Code != http.StatusForbidden {
		t.Fatalf("add entry by non-owner status = %d, want 403, body = %s", entryW.Code, entryW.Body.String())
	}

	visC, visW := newTestContext(t, http.MethodPut, "/users/"+intruder+"/wishlists/"+created.ID+"/visibility", visibilityRequest{Visibility: "public"},
		gin.Params{{Key: "user_id", Value: intruder}, {Key: "wishlist_id", Value: created.ID}}, intruder)
	setWishlistVisibility(visC)
	if visW.Code != http.StatusForbidden {
		t.Fatalf("set visibility by non-owner status = %d, want 403, body = %s", visW.Code, visW.Body.String())
	}

	stillThereC, stillThereW := newTestContext(t, http.MethodGet, "/users/"+owner+"/wishlists/"+created.ID, nil,
		gin.Params{{Key: "user_id", Value: owner}, {Key: "wishlist_id", Value: created.ID}}, owner)
	getWishlistByID(stillThereC)
	if stillThereW.Code != http.StatusOK {
		t.Fatalf("owner get after intruder attempts status = %d, want 200", stillThereW.Code)
	}
	var stillThere objvo.WishlistVO
	if err := json.Unmarshal(stillThereW.Body.Bytes(), &stillThere); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if stillThere.Visibility != "private" || len(stillThere.Entries) != 0 {
		t.Fatalf("intruder mutated the wishlist despite 403s: %+v", stillThere)
	}
}
