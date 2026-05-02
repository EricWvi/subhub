package subscription

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
	mux.HandleFunc("/subscriptions/clash-configs", h.handleClashConfigSubscriptions)
	mux.HandleFunc("/subscriptions/clash-configs/", h.handleClashConfigSubscriptionByID)
	mux.HandleFunc("/subscriptions/proxy-providers", h.handleProxyProviderSubscriptions)
	mux.HandleFunc("/subscriptions/proxy-providers/", h.handleProxyProviderSubscriptionByID)
	mux.HandleFunc("/subscriptions/rule-providers", h.handleRuleProviderSubscriptions)
	mux.HandleFunc("/subscriptions/rule-providers/", h.handleRuleProviderSubscriptionByID)
}

func (h *Handler) handleClashConfigSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listClashConfigs(w, r)
	case http.MethodPost:
		h.createClashConfig(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleClashConfigSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/subscriptions/clash-configs/")
	if path == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid subscription id", http.StatusBadRequest)
		return
	}

	if len(parts) >= 2 && parts[1] == "content" {
		if r.Method == http.MethodGet {
			h.getClashConfigContent(w, r, id)
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
		h.getClashConfig(w, r, id)
	case http.MethodPut:
		h.updateClashConfig(w, r, id)
	case http.MethodDelete:
		h.deleteClashConfig(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleProxyProviderSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProxyProviders(w, r)
	case http.MethodPost:
		h.createProxyProvider(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleProxyProviderSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/subscriptions/proxy-providers/")
	if path == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid subscription id", http.StatusBadRequest)
		return
	}

	if len(parts) >= 2 && parts[1] == "content" {
		if r.Method == http.MethodGet {
			h.getProxyProviderContent(w, r, id)
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
		h.getProxyProvider(w, r, id)
	case http.MethodPut:
		h.updateProxyProvider(w, r, id)
	case http.MethodDelete:
		h.deleteProxyProvider(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleRuleProviderSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listRuleProviders(w, r)
	case http.MethodPost:
		h.createRuleProvider(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleRuleProviderSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/subscriptions/rule-providers/")
	if path == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid subscription id", http.StatusBadRequest)
		return
	}

	if len(parts) >= 2 && parts[1] == "content" {
		if r.Method == http.MethodGet {
			h.getRuleProviderContent(w, r, id)
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
		h.getRuleProvider(w, r, id)
	case http.MethodPut:
		h.updateRuleProvider(w, r, id)
	case http.MethodDelete:
		h.deleteRuleProvider(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listClashConfigs(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /subscriptions/clash-configs")
	subs, err := h.service.ListClashConfigs(r.Context())
	if err != nil {
		log.Printf("[API] List clash config subscriptions failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if subs == nil {
		subs = []ClashConfigSubscription{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscriptions": subs})
}

func (h *Handler) createClashConfig(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /subscriptions/clash-configs")
	var in CreateClashConfigSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sub, err := h.service.CreateClashConfig(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrProvidersRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("[API] Create clash config failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) getClashConfig(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /subscriptions/clash-configs/%d", id)
	sub, err := h.service.GetClashConfigByID(r.Context(), id)
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

func (h *Handler) updateClashConfig(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] PUT /subscriptions/clash-configs/%d", id)
	var in UpdateClashConfigSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sub, err := h.service.UpdateClashConfig(r.Context(), id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrProvidersRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNotFound):
			http.Error(w, "subscription not found", http.StatusNotFound)
		default:
			log.Printf("[API] Update clash config failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) deleteClashConfig(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] DELETE /subscriptions/clash-configs/%d", id)
	err := h.service.DeleteClashConfig(r.Context(), id)
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

func (h *Handler) getClashConfigContent(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /subscriptions/clash-configs/%d/content", id)
	content, err := h.service.BuildClashConfigContent(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "subscription not found", http.StatusNotFound)
		} else {
			log.Printf("[API] Build clash config content failed: %v", err)
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

func (h *Handler) listProxyProviders(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /subscriptions/proxy-providers")
	subs, err := h.service.ListProxyProviders(r.Context())
	if err != nil {
		log.Printf("[API] List proxy provider subscriptions failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if subs == nil {
		subs = []ProxyProviderSubscription{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscriptions": subs})
}

func (h *Handler) createProxyProvider(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /subscriptions/proxy-providers")
	var in CreateProxyProviderSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sub, err := h.service.CreateProxyProvider(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrProvidersRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("[API] Create proxy provider failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) getProxyProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /subscriptions/proxy-providers/%d", id)
	sub, err := h.service.GetProxyProviderByID(r.Context(), id)
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

func (h *Handler) updateProxyProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] PUT /subscriptions/proxy-providers/%d", id)
	var in UpdateProxyProviderSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sub, err := h.service.UpdateProxyProvider(r.Context(), id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrProvidersRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNotFound):
			http.Error(w, "subscription not found", http.StatusNotFound)
		default:
			log.Printf("[API] Update proxy provider failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) deleteProxyProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] DELETE /subscriptions/proxy-providers/%d", id)
	err := h.service.DeleteProxyProvider(r.Context(), id)
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

func (h *Handler) listRuleProviders(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /subscriptions/rule-providers")
	subs, err := h.service.ListRuleProviders(r.Context())
	if err != nil {
		log.Printf("[API] List rule provider subscriptions failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if subs == nil {
		subs = []RuleProviderSubscription{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscriptions": subs})
}

func (h *Handler) createRuleProvider(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /subscriptions/rule-providers")
	var in CreateRuleProviderSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sub, err := h.service.CreateRuleProvider(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrProvidersRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("[API] Create rule provider failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) getRuleProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /subscriptions/rule-providers/%d", id)
	sub, err := h.service.GetRuleProviderByID(r.Context(), id)
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

func (h *Handler) updateRuleProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] PUT /subscriptions/rule-providers/%d", id)
	var in UpdateRuleProviderSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sub, err := h.service.UpdateRuleProvider(r.Context(), id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrProvidersRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNotFound):
			http.Error(w, "subscription not found", http.StatusNotFound)
		default:
			log.Printf("[API] Update rule provider failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

func (h *Handler) deleteRuleProvider(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] DELETE /subscriptions/rule-providers/%d", id)
	err := h.service.DeleteRuleProvider(r.Context(), id)
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

func (h *Handler) getProxyProviderContent(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /subscriptions/proxy-providers/%d/content", id)
	content, err := h.service.BuildProxyProviderContent(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "subscription not found", http.StatusNotFound)
		} else {
			log.Printf("[API] Build proxy provider content failed: %v", err)
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

func (h *Handler) getRuleProviderContent(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /subscriptions/rule-providers/%d/content", id)
	content, err := h.service.BuildRuleProviderContent(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "subscription not found", http.StatusNotFound)
		} else {
			log.Printf("[API] Build rule provider content failed: %v", err)
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
