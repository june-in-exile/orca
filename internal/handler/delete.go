package handler

import (
	"log/slog"
	"net/http"

	"github.com/anthropics/paylock/internal/model"
	"github.com/anthropics/paylock/internal/suiauth"
)

type Delete struct {
	videos   *model.VideoStore
	verifier SigVerifier
	clock    suiauth.Clock
}

func NewDelete(videos *model.VideoStore, verifier SigVerifier, clock suiauth.Clock) *Delete {
	return &Delete{videos: videos, verifier: verifier, clock: clock}
}

func (h *Delete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing video id",
		})
		return
	}

	video, ok := h.videos.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "video not found",
		})
		return
	}

	if video.Creator != "" {
		auth := extractAndVerifyWalletAuth(r, h.verifier, h.clock, "delete", id)
		if auth.err != "" {
			writeJSON(w, auth.status, map[string]string{"error": auth.err})
			return
		}
		if !verifyOwnership(video, auth.address) {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "forbidden: wallet address does not match video creator",
			})
			return
		}
	}

	h.videos.Delete(id)
	slog.Info("video deleted", "id", id)
	writeJSON(w, http.StatusOK, map[string]string{
		"id":     id,
		"status": "deleted",
	})
}
