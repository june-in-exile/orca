package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/anthropics/orca/internal/config"
	"github.com/anthropics/orca/internal/model"
	"github.com/anthropics/orca/internal/processor"
	"github.com/anthropics/orca/internal/storage"
)

type Upload struct {
	store  storage.Backend
	proc   *processor.Processor
	videos *model.VideoStore
	cfg    *config.Config
}

func NewUpload(store storage.Backend, proc *processor.Processor, videos *model.VideoStore, cfg *config.Config) *Upload {
	return &Upload{store: store, proc: proc, videos: videos, cfg: cfg}
}

func (h *Upload) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxFileSize)

	file, header, err := r.FormFile("video")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "failed to read video file: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate file size
	if err := processor.ValidateSize(header.Size, h.cfg.MaxFileSize); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Read header bytes for magic byte validation
	headerBytes := make([]byte, 12)
	n, err := io.ReadFull(file, headerBytes)
	if err != nil || n < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "file too small or unreadable",
		})
		return
	}

	if err := processor.ValidateMagicBytes(bytes.NewReader(headerBytes[:n])); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid file format: only MP4 files are accepted",
		})
		return
	}

	// Generate video ID
	id := generateID()

	// Stream file to disk: prepend the already-read header bytes
	reader := io.MultiReader(bytes.NewReader(headerBytes[:n]), file)
	filePath, err := h.store.SaveUpload(id, reader)
	if err != nil {
		slog.Error("failed to save upload", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to save video",
		})
		return
	}

	// Register video and start background processing
	h.videos.Create(id)

	go h.processVideo(id, filePath)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"id":     id,
		"status": model.StatusProcessing,
	})
}

func (h *Upload) processVideo(id, filePath string) {
	ctx := context.Background()

	// Validate with ffprobe
	duration, err := h.proc.Probe(filePath)
	if err != nil {
		slog.Error("ffprobe validation failed", "id", id, "error", err)
		h.videos.SetFailed(id, "video validation failed: "+err.Error())
		return
	}

	// Segment with ffmpeg
	outputDir := h.store.OutputDir(id)
	if err := h.proc.Segment(ctx, filePath, outputDir); err != nil {
		slog.Error("ffmpeg segmentation failed", "id", id, "error", err)
		h.videos.SetFailed(id, "video processing failed: "+err.Error())
		return
	}

	h.videos.SetReady(id, duration)
	slog.Info("video ready", "id", id, "duration", duration)
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
