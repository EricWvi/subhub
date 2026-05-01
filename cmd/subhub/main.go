package main

import (
	"log"
	"net/http"

	"github.com/EricWvi/subhub/internal/config"
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/store"
)

func main() {
	cfg := config.Load()
	db := store.MustOpen(cfg.DatabasePath)
	defer db.Close()

	repo := provider.NewRepository(db)
	svc := provider.NewService(repo)
	handler := provider.NewHandler(svc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}
