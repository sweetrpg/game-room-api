package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sweetrpg/game-room-data.go/data"
	objvo "github.com/sweetrpg/game-room-objects.go/vo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUpdateLibraryEntryTitle(t *testing.T) {
	setupTestDB(t)
	userID := primitive.NewObjectID().Hex()

	_, err := data.AddLibraryEntry(t.Context(), userID, "vol-1", "Old Title")
	if err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	c, w := newTestContext(t, http.MethodPut, "/users/"+userID+"/library/entries/vol-1/title", titleRequest{Title: "New Title"}, gin.Params{{Key: "user_id", Value: userID}, {Key: "volume_id", Value: "vol-1"}}, userID)
	updateLibraryEntryTitle(c)
	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", w.Code, w.Body.String())
	}

	var vo objvo.LibraryVO
	if err := json.Unmarshal(w.Body.Bytes(), &vo); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(vo.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(vo.Entries))
	}
	if vo.Entries[0].VolumeTitle != "New Title" {
		t.Errorf("title = %q, want %q", vo.Entries[0].VolumeTitle, "New Title")
	}
	if vo.Entries[0].VolumeID != "vol-1" {
		t.Errorf("volume_id = %q, want vol-1", vo.Entries[0].VolumeID)
	}
}

func TestUpdateLibraryEntryTitleMissingEntry(t *testing.T) {
	setupTestDB(t)
	userID := primitive.NewObjectID().Hex()

	_, err := data.AddLibraryEntry(t.Context(), userID, "vol-1", "Maus I")
	if err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	c, w := newTestContext(t, http.MethodPut, "/users/"+userID+"/library/entries/vol-999/title", titleRequest{Title: "Nope"}, gin.Params{{Key: "user_id", Value: userID}, {Key: "volume_id", Value: "vol-999"}}, userID)
	updateLibraryEntryTitle(c)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestUpdateLibraryEntryTitleMissingBody(t *testing.T) {
	setupTestDB(t)
	userID := primitive.NewObjectID().Hex()

	c, w := newTestContext(t, http.MethodPut, "/users/"+userID+"/library/entries/vol-1/title", nil, gin.Params{}, userID)
	updateLibraryEntryTitle(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
