package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/EricWvi/subhub/internal/config"
	"github.com/EricWvi/subhub/internal/fetch"
	"github.com/EricWvi/subhub/internal/group"
	"github.com/EricWvi/subhub/internal/output"
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/refresh"
	"github.com/EricWvi/subhub/internal/store"
)

func main() {
	cfg := config.Load()
	db := store.MustOpen(cfg.DatabasePath)
	defer db.Close()

	repo := provider.NewRepository(db)
	svc := provider.NewService(repo)
	handler := provider.NewHandler(svc)

	fetcher := fetch.NewClient(cfg.UpstreamRequestTimeout)
	refreshSvc := refresh.NewService(repo, fetcher)
	handler.SetRefresher(refreshSvc.RefreshProvider)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	outputHandler := output.NewHandler(repo, "tests/fixtures/template.yaml")
	outputHandler.RegisterRoutes(mux)

	groupRepo := group.NewRepository(db)
	groupSvc := group.NewService(groupRepo)
	groupHandler := group.NewHandler(groupSvc)
	groupHandler.RegisterRoutes(mux)

	scheduler := refresh.NewScheduler(repo, refreshSvc.RefreshProvider, time.Minute)
	go scheduler.Start(context.Background())

	log.Printf("[MAIN] Starting SubHub on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}
