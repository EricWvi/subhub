package provider

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RefreshProviderFunc func(ctx context.Context, providerID int64) error

type Handler struct {
	service   *Service
	refresher RefreshProviderFunc
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SetRefresher(f RefreshProviderFunc) {
	h.refresher = f
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/providers", h.handleProviders)
	mux.HandleFunc("/providers/", h.handleProviderByID)
}

func (h *Handler) handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProviders(w, r)
	case http.MethodPost:
		h.createProvider(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleProviderByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/providers/")
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid provider id", http.StatusBadRequest)
		return
	}

	if len(parts) >= 2 {
		switch parts[1] {
		case "refresh":
			if r.Method == http.MethodPost {
				h.refreshProvider(w, r, id)
				return
			}
		case "snapshot":
			if r.Method == http.MethodGet {
				h.getSnapshot(w, r, id)
				return
			}
		case "nodes":
			if r.Method == http.MethodGet {
				h.getNodes(w, r, id)
				return
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getProvider(w, r, id)
	case http.MethodPut:
		h.updateProvider(w, r, id)
	case http.MethodDelete:
		h.deleteProvider(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) refreshProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] POST /providers/%d/refresh", id)
	if h.refresher == nil {
		log.Printf("[API] Refresh not configured")
		http.Error(w, "refresh not configured", http.StatusServiceUnavailable)
		return
	}
	if err := h.refresher(r.Context(), id); err != nil {
		log.Printf("[API] Refresh failed for provider %d: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "provider not found", http.StatusNotFound)
		} else {
			var rfe *RefreshFailedError
			if errors.As(err, &rfe) {
				http.Error(w, err.Error(), http.StatusBadGateway)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		return
	}
	log.Printf("[API] Refresh success for provider %d", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listProviders(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /providers")
	providers, err := h.service.List(r.Context())
	if err != nil {
		log.Printf("[API] List providers failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if providers == nil {
		providers = []Provider{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"providers": providers})
}

func (h *Handler) createProvider(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /providers")
	var in CreateProviderInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Printf("[API] Decode create provider input failed: %v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	p, err := h.service.Create(r.Context(), in)
	if err != nil {
		log.Printf("[API] Create provider failed: %v", err)
		switch {
		case errors.Is(err, ErrRefreshIntervalTooShort):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrInvalidURL):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, errInvalidAbbrev):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("[API] Created provider %d (%s)", p.ID, p.Name)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"provider": p})

	// Trigger initial fetch
	if h.refresher != nil {
		go func() {
			log.Printf("[BG] Triggering initial fetch for provider %d", p.ID)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if err := h.refresher(ctx, p.ID); err != nil {
				log.Printf("[BG] Initial fetch failed for provider %d: %v", p.ID, err)
			} else {
				log.Printf("[BG] Initial fetch success for provider %d", p.ID)
			}
		}()
	}
}

func (h *Handler) getProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /providers/%d", id)
	p, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("[API] Get provider %d failed: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "provider not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"provider": p})
}

func (h *Handler) updateProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] PUT /providers/%d", id)
	var in UpdateProviderInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Printf("[API] Decode update provider input failed: %v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	p, err := h.service.Update(r.Context(), id, in)
	if err != nil {
		log.Printf("[API] Update provider %d failed: %v", id, err)
		switch {
		case errors.Is(err, ErrRefreshIntervalTooShort):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrInvalidURL):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, errInvalidAbbrev):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("[API] Updated provider %d", id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"provider": p})
}

func (h *Handler) deleteProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] DELETE /providers/%d", id)
	err := h.service.Delete(r.Context(), id)
	if err != nil {
		log.Printf("[API] Delete provider %d failed: %v", id, err)
		switch {
		case errors.Is(err, ErrSubscriptionProviderRef):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, ErrNotFound):
			http.Error(w, "provider not found", http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("[API] Deleted provider %d", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getSnapshot(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /providers/%d/snapshot", id)
	s, err := h.service.GetLatestSnapshot(r.Context(), id)
	if err != nil {
		log.Printf("[API] Get snapshot for provider %d failed: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "snapshot not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"snapshot": s})
}

func (h *Handler) getNodes(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /providers/%d/nodes", id)
	nodes, err := h.service.ListNodes(r.Context(), id)
	if err != nil {
		log.Printf("[API] List nodes for provider %d failed: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if nodes == nil {
		nodes = []ProxyNode{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"nodes": nodes})
}
