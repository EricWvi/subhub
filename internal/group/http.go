package group

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/proxy-groups", h.handleGroups)
	mux.HandleFunc("/proxy-groups/", h.handleGroupByID)
}

func (h *Handler) handleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listGroups(w, r)
	case http.MethodPost:
		h.createGroup(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGroupByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/proxy-groups/")
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid proxy group id", http.StatusBadRequest)
		return
	}

	if len(parts) >= 2 {
		switch parts[1] {
		case "nodes":
			if r.Method == http.MethodGet {
				h.getGroupNodes(w, r, id)
				return
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getGroup(w, r, id)
	case http.MethodPut:
		h.updateGroup(w, r, id)
	case http.MethodDelete:
		h.deleteGroup(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /proxy-groups")
	groups, err := h.service.List(r.Context())
	if err != nil {
		log.Printf("[API] List proxy groups failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []ProxyGroup{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"groups": groups})
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /proxy-groups")
	var in CreateGroupInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Printf("[API] Decode create proxy group input failed: %v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	g, err := h.service.Create(r.Context(), in)
	if err != nil {
		log.Printf("[API] Create proxy group failed: %v", err)
		if errors.Is(err, ErrNameRequired) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("[API] Created proxy group %d (%s)", g.ID, g.Name)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"group": g})
}

func (h *Handler) getGroup(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /proxy-groups/%d", id)
	g, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("[API] Get proxy group %d failed: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "proxy group not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"group": g})
}

func (h *Handler) updateGroup(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] PUT /proxy-groups/%d", id)
	var in UpdateGroupInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Printf("[API] Decode update proxy group input failed: %v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	g, err := h.service.Update(r.Context(), id, in)
	if err != nil {
		log.Printf("[API] Update proxy group %d failed: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "proxy group not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("[API] Updated proxy group %d", id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"group": g})
}

func (h *Handler) deleteGroup(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] DELETE /proxy-groups/%d", id)
	err := h.service.Delete(r.Context(), id)
	if err != nil {
		log.Printf("[API] Delete proxy group %d failed: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[API] Deleted proxy group %d", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getGroupNodes(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /proxy-groups/%d/nodes", id)
	nodes, err := h.service.ListNodes(r.Context(), id)
	if err != nil {
		log.Printf("[API] Get nodes for proxy group %d failed: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "proxy group not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if nodes == nil {
		nodes = []ProxyNodeView{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"nodes": nodes})
}
