package handler

import (
	"net/http"

	"github.com/anthropics/paylock/internal/model"
)

type VideoByObject struct {
	videos *model.VideoStore
}

func NewVideoByObject(videos *model.VideoStore) *VideoByObject {
	return &VideoByObject{videos: videos}
}

// ServeHTTP returns video metadata looked up by sui_object_id.
func (h *VideoByObject) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	objectID := r.PathValue("object_id")
	if objectID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing object_id",
		})
		return
	}

	video, ok := h.videos.GetBySuiObjectID(objectID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "video not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, video)
}
