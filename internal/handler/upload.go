package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/anthropics/orca/internal/config"
	"github.com/anthropics/orca/internal/model"
	"github.com/anthropics/orca/internal/processor"
	"github.com/anthropics/orca/internal/walrus"
)

type Upload struct {
	walrus *walrus.Client
	videos *model.VideoStore
	cfg    *config.Config
}

func NewUpload(w *walrus.Client, videos *model.VideoStore, cfg *config.Config) *Upload {
	return &Upload{walrus: w, videos: videos, cfg: cfg}
}

func (h *Upload) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxFileSize)

	file, header, err := r.FormFile("video")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "failed to read video file: " + err.Error(),
		})
		return
	}
	defer file.Close()

	if err := processor.ValidateSize(header.Size, h.cfg.MaxFileSize); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
			"error": err.Error(),
		})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "failed to read video file",
		})
		return
	}

	if err := processor.ValidateMagicBytes(bytes.NewReader(data)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid file format: only MP4 files are accepted",
		})
		return
	}

	id := generateID()
	title := r.FormValue("title")
	if title == "" {
		title = id
	}

	h.videos.Create(id, title)

	go h.uploadToWalrus(id, data)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"id":     id,
		"status": model.StatusProcessing,
	})
}

func (h *Upload) uploadToWalrus(id string, data []byte) {
	blobID, err := h.walrus.Store(data, h.cfg.WalrusEpochs)
	if err != nil {
		slog.Error("walrus upload failed", "id", id, "error", err)
		h.videos.SetFailed(id, "upload to Walrus failed: "+err.Error())
		return
	}

	blobURL := h.walrus.BlobURL(blobID)
	h.videos.SetReady(id, blobID, blobURL)
	slog.Info("video uploaded to walrus", "id", id, "blob_id", blobID)
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
