package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/paylock/internal/model"
)

func TestVideoByObject_Found(t *testing.T) {
	store := mustNewVideoStore(t)
	store.Create("abc123", "Test Video", 0, "0xCreator")
	store.SetReady("abc123", "tb", "https://agg/v1/blobs/tb", "pb", "https://agg/v1/blobs/pb", "fb", "https://agg/v1/blobs/fb")
	store.SetSuiObjectID("abc123", "0xOBJ999", "", "")

	h := NewVideoByObject(store)
	req := httptest.NewRequest(http.MethodGet, "/api/videos/by-object/0xOBJ999", nil)
	req.SetPathValue("object_id", "0xOBJ999")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var video model.Video
	if err := json.NewDecoder(rec.Body).Decode(&video); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if video.ID != "abc123" {
		t.Errorf("expected id abc123, got %s", video.ID)
	}
	if video.SuiObjectID != "0xOBJ999" {
		t.Errorf("expected sui_object_id 0xOBJ999, got %s", video.SuiObjectID)
	}
}

func TestVideoByObject_NotFound(t *testing.T) {
	store := mustNewVideoStore(t)
	h := NewVideoByObject(store)

	req := httptest.NewRequest(http.MethodGet, "/api/videos/by-object/0xNONE", nil)
	req.SetPathValue("object_id", "0xNONE")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestVideoByObject_MissingID(t *testing.T) {
	store := mustNewVideoStore(t)
	h := NewVideoByObject(store)

	req := httptest.NewRequest(http.MethodGet, "/api/videos/by-object/", nil)
	req.SetPathValue("object_id", "")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
