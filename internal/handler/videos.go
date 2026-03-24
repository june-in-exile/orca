package handler

import (
	"net/http"
	"sort"

	"github.com/anthropics/paylock/internal/model"
)

type Videos struct {
	videos *model.VideoStore
}

func NewVideos(videos *model.VideoStore) *Videos {
	return &Videos{videos: videos}
}

func (h *Videos) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	list := h.videos.List()

	if creator := r.URL.Query().Get("creator"); creator != "" {
		filtered := make([]model.Video, 0, len(list))
		for _, v := range list {
			if v.Creator == creator {
				filtered = append(filtered, v)
			}
		}
		list = filtered
	}

	// Sort by created_at descending (newest first)
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt > list[j].CreatedAt
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"videos": list,
	})
}
