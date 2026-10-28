package subscription

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (h *Handler) handleNodeProviderSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listNodeProviders(w, r)
	case http.MethodPost:
		h.createNodeProvider(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleNodeProviderSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/subscriptions/node-providers/")
	if path == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid subscription id", http.StatusBadRequest)
		return
	}

	if len(parts) == 2 && parts[1] == "content" {
		if r.Method == http.MethodGet {
			h.getNodeProviderContent(w, r, id)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(parts) >= 2 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getNodeProvider(w, r, id)
	case http.MethodPut:
		h.updateNodeProvider(w, r, id)
	case http.MethodDelete:
		h.deleteNodeProvider(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listNodeProviders(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /subscriptions/node-providers")
	subs, err := h.service.ListNodeProviders(r.Context())
	if err != nil {
		log.Printf("[API] List node provider subscriptions failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if subs == nil {
		subs = []NodeProviderSubscription{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscriptions": subs})
}

func (h *Handler) createNodeProvider(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /subscriptions/node-providers")
	var in CreateNodeProviderSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sub, err := h.service.CreateNodeProvider(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNodesRequired), errors.Is(err, ErrInvalidCustomNode), errors.Is(err, ErrCustomNodeNotFound), errors.Is(err, ErrDuplicateNodeSelection):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("[API] Create node provider failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) getNodeProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /subscriptions/node-providers/%d", id)
	sub, err := h.service.GetNodeProviderByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "subscription not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) updateNodeProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] PUT /subscriptions/node-providers/%d", id)
	var in UpdateNodeProviderSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sub, err := h.service.UpdateNodeProvider(r.Context(), id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNodesRequired), errors.Is(err, ErrInvalidCustomNode), errors.Is(err, ErrCustomNodeNotFound), errors.Is(err, ErrDuplicateNodeSelection):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNotFound):
			http.Error(w, "subscription not found", http.StatusNotFound)
		default:
			log.Printf("[API] Update node provider failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) deleteNodeProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] DELETE /subscriptions/node-providers/%d", id)
	err := h.service.DeleteNodeProvider(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "subscription not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getNodeProviderContent(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /subscriptions/node-providers/%d/content", id)
	content, err := h.service.BuildNodeProviderContent(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "subscription not found", http.StatusNotFound)
		} else {
			log.Printf("[API] Build node provider content failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if content.SubscriptionUserinfo != "" {
		w.Header().Set("Subscription-Userinfo", content.SubscriptionUserinfo)
	}
	w.Header().Set("Content-Type", content.ContentType)
	w.Write([]byte(content.Body))
}
