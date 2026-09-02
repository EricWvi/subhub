package node

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
	mux.HandleFunc("/nodes", h.handleNodes)
	mux.HandleFunc("/nodes/", h.handleNodeByID)
}

func (h *Handler) handleNodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listNodes(w, r)
	case http.MethodPost:
		h.createNode(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleNodeByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/nodes/")
	parts := strings.Split(path, "/")
	if len(parts) != 1 || parts[0] == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid node id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getNode(w, r, id)
	case http.MethodPut:
		h.updateNode(w, r, id)
	case http.MethodDelete:
		h.deleteNode(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listNodes(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /nodes")
	nodes, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if nodes == nil {
		nodes = []Node{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"nodes": nodes})
}

func (h *Handler) createNode(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /nodes")
	var in CreateNodeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	n, err := h.service.Create(r.Context(), in)
	if err != nil {
		handleNodeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"node": n})
}

func (h *Handler) getNode(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /nodes/%d", id)
	n, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleNodeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"node": n})
}

func (h *Handler) updateNode(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] PUT /nodes/%d", id)
	var in UpdateNodeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	n, err := h.service.Update(r.Context(), id, in)
	if err != nil {
		handleNodeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"node": n})
}

func (h *Handler) deleteNode(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] DELETE /nodes/%d", id)
	if err := h.service.Delete(r.Context(), id); err != nil {
		handleNodeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleNodeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNameRequired), errors.Is(err, ErrConfigRequired), errors.Is(err, ErrInvalidConfig):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrNotFound):
		http.Error(w, "node not found", http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
