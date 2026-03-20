package model

import (
	"testing"
)

func TestVideoStore_CreateAndGet(t *testing.T) {
	store := NewVideoStore()

	store.Create("test-1", "My Video")

	v, ok := store.Get("test-1")
	if !ok {
		t.Fatal("expected video to exist")
	}
	if v.ID != "test-1" {
		t.Errorf("expected id test-1, got %s", v.ID)
	}
	if v.Title != "My Video" {
		t.Errorf("expected title My Video, got %s", v.Title)
	}
	if v.Status != StatusProcessing {
		t.Errorf("expected status processing, got %s", v.Status)
	}
}

func TestVideoStore_GetNotFound(t *testing.T) {
	store := NewVideoStore()

	_, ok := store.Get("nonexistent")
	if ok {
		t.Fatal("expected video not to exist")
	}
}

func TestVideoStore_SetReady(t *testing.T) {
	store := NewVideoStore()
	store.Create("test-1", "My Video")

	store.SetReady("test-1", "blob123", "https://aggregator/v1/blobs/blob123")

	v, _ := store.Get("test-1")
	if v.Status != StatusReady {
		t.Errorf("expected status ready, got %s", v.Status)
	}
	if v.BlobID != "blob123" {
		t.Errorf("expected blob_id blob123, got %s", v.BlobID)
	}
	if v.BlobURL != "https://aggregator/v1/blobs/blob123" {
		t.Errorf("expected blob_url, got %s", v.BlobURL)
	}
}

func TestVideoStore_SetFailed(t *testing.T) {
	store := NewVideoStore()
	store.Create("test-1", "My Video")

	store.SetFailed("test-1", "something went wrong")

	v, _ := store.Get("test-1")
	if v.Status != StatusFailed {
		t.Errorf("expected status failed, got %s", v.Status)
	}
	if v.Error != "something went wrong" {
		t.Errorf("expected error message, got %s", v.Error)
	}
}

func TestVideoStore_GetReturnsImmutableCopy(t *testing.T) {
	store := NewVideoStore()
	store.Create("test-1", "My Video")

	v1, _ := store.Get("test-1")
	v1.Status = StatusReady // mutate the copy

	v2, _ := store.Get("test-1")
	if v2.Status != StatusProcessing {
		t.Error("mutation of returned copy should not affect store")
	}
}

func TestVideoStore_List(t *testing.T) {
	store := NewVideoStore()
	store.Create("a", "Title A")
	store.Create("b", "Title B")

	list := store.List()
	if len(list) != 2 {
		t.Errorf("expected 2 videos, got %d", len(list))
	}
}

func TestVideoStore_Delete(t *testing.T) {
	store := NewVideoStore()
	store.Create("test-1", "My Video")

	if !store.Delete("test-1") {
		t.Fatal("expected delete to return true")
	}

	_, ok := store.Get("test-1")
	if ok {
		t.Fatal("expected video to be deleted")
	}

	if store.Delete("test-1") {
		t.Fatal("expected delete of nonexistent to return false")
	}
}
