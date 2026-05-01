package main

import (
	"io"
	"log"
	"net/http"

	"github.com/EricWvi/subhub/internal/config"
	"github.com/EricWvi/subhub/internal/store"
)

func main() {
	cfg := config.Load()
	db := store.MustOpen(cfg.DatabasePath)
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/providers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"providers":[]}`)
	})

	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}
