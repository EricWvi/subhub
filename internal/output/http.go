package output

import (
	"io"
	"net/http"

	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/render"
)

type Handler struct {
	providers    *provider.Repository
	templatePath string
}

func NewHandler(providers *provider.Repository, templatePath string) *Handler {
	return &Handler{providers: providers, templatePath: templatePath}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/subscriptions/mihomo", h.ServeHTTP)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	nodes, err := h.providers.ListLatestNodes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out, err := render.MihomoTemplate(h.templatePath, nodes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	io.WriteString(w, out)
}
