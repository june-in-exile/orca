package handler

import (
	"net/http"

	"github.com/anthropics/paylock/internal/model"
)

type Video struct {
	videos *model.VideoStore
}

func NewVideo(videos *model.VideoStore) *Video {
	return &Video{videos: videos}
}

func (h *Video) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing video id",
		})
		return
	}

	video, ok := h.videos.Get(id)
	if !ok {
		video, ok = h.videos.GetBySuiObjectID(id)
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "video not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, video)
}
