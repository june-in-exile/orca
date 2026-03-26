package handler

import (
	"encoding/json"
	"net/http"

	"github.com/anthropics/paylock/internal/model"
	"github.com/anthropics/paylock/internal/suiauth"
)

type SetSuiObject struct {
	videos   *model.VideoStore
	walrus   Storer
	verifier SigVerifier
	clock    suiauth.Clock
	sessions *SessionStore
}

func NewSetSuiObject(videos *model.VideoStore, walrus Storer, verifier SigVerifier, clock suiauth.Clock, sessions *SessionStore) *SetSuiObject {
	return &SetSuiObject{videos: videos, walrus: walrus, verifier: verifier, clock: clock, sessions: sessions}
}

func (h *SetSuiObject) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing video id",
		})
		return
	}

	var body struct {
		SuiObjectID string `json:"sui_object_id"`
		FullBlobID  string `json:"full_blob_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if body.SuiObjectID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "sui_object_id is required",
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
		if token := r.Header.Get("X-Session-Token"); token != "" && h.sessions != nil {
			addr, ok := h.sessions.Validate(token)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "session expired or invalid",
				})
				return
			}
			if !verifyOwnership(video, addr) {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"error": "forbidden: wallet address does not match video creator",
				})
				return
			}
		} else {
			auth := extractAndVerifyWalletAuth(r, h.verifier, h.clock, "update", id)
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
	}

	if video.SuiObjectID != "" && video.SuiObjectID != body.SuiObjectID {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "sui object id already set to a different value",
		})
		return
	}

	var fullBlobURL string
	if body.FullBlobID != "" {
		fullBlobURL = h.walrus.BlobURL(body.FullBlobID)
	}

	if !h.videos.SetSuiObjectID(id, body.SuiObjectID, body.FullBlobID, fullBlobURL) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "video not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":        "ok",
		"sui_object_id": body.SuiObjectID,
	})
}
