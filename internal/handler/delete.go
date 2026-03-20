package handler

import (
	"log/slog"
	"net/http"

	"github.com/anthropics/orca/internal/model"
)

type Delete struct {
	videos *model.VideoStore
}

func NewDelete(videos *model.VideoStore) *Delete {
	return &Delete{videos: videos}
}

func (h *Delete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing video id",
		})
		return
	}

	if !h.videos.Delete(id) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "video not found",
		})
		return
	}

	slog.Info("video deleted", "id", id)
	writeJSON(w, http.StatusOK, map[string]string{
		"id":     id,
		"status": "deleted",
	})
}
