package rule

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
	mux.HandleFunc("/rules", h.handleRules)
	mux.HandleFunc("/rules/", h.handleRuleByID)
}

func (h *Handler) handleRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listRules(w, r)
	case http.MethodPost:
		h.createRule(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleRuleByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/rules/")
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid rule id", http.StatusBadRequest)
		return
	}

	if len(parts) >= 2 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getRule(w, r, id)
	case http.MethodPut:
		h.updateRule(w, r, id)
	case http.MethodDelete:
		h.deleteRule(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listRules(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /rules")

	in := parseListRulesInput(r)

	result, err := h.service.List(r.Context(), in)
	if err != nil {
		log.Printf("[API] List rules failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func parseListRulesInput(r *http.Request) ListRulesInput {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	search := r.URL.Query().Get("search")
	return ListRulesInput{Page: page, PageSize: pageSize, Search: search}
}

func (h *Handler) createRule(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /rules")
	var in CreateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Printf("[API] Decode create rule input failed: %v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rule, err := h.service.Create(r.Context(), in)
	if err != nil {
		log.Printf("[API] Create rule failed: %v", err)
		if errors.Is(err, ErrRuleTypeRequired) || errors.Is(err, ErrPatternRequired) ||
			errors.Is(err, ErrProxyGroupRequired) || errors.Is(err, ErrInvalidProxyGroup) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("[API] Created rule %d", rule.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"rule": rule})
}

func (h *Handler) getRule(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] GET /rules/%d", id)
	rule, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("[API] Get rule %d failed: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "rule not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"rule": rule})
}

func (h *Handler) updateRule(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] PUT /rules/%d", id)
	var in UpdateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Printf("[API] Decode update rule input failed: %v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rule, err := h.service.Update(r.Context(), id, in)
	if err != nil {
		log.Printf("[API] Update rule %d failed: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "rule not found", http.StatusNotFound)
		} else if errors.Is(err, ErrRuleTypeRequired) || errors.Is(err, ErrPatternRequired) ||
			errors.Is(err, ErrProxyGroupRequired) || errors.Is(err, ErrInvalidProxyGroup) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("[API] Updated rule %d", id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"rule": rule})
}

func (h *Handler) deleteRule(w http.ResponseWriter, r *http.Request, id int64) {
	log.Printf("[API] DELETE /rules/%d", id)
	err := h.service.Delete(r.Context(), id)
	if err != nil {
		log.Printf("[API] Delete rule %d failed: %v", id, err)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "rule not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("[API] Deleted rule %d", id)
	w.WriteHeader(http.StatusNoContent)
}
