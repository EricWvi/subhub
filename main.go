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
	"github.com/EricWvi/subhub/internal/rule"
	"github.com/EricWvi/subhub/internal/store"
	"github.com/EricWvi/subhub/internal/subscription"
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

	groupRepo := group.NewRepository(db)
	groupSvc := group.NewService(groupRepo)
	groupHandler := group.NewHandler(groupSvc)

	ruleRepo := rule.NewRepository(db)
	ruleSvc := rule.NewService(ruleRepo)
	ruleHandler := rule.NewHandler(ruleSvc)

	outputHandler := output.NewHandler(repo, ruleRepo, "data/template.yaml")

	subscriptionRepo := subscription.NewRepository(db)
	subscriptionSvc := subscription.NewService(subscriptionRepo, repo, groupSvc, ruleRepo, "data/client_sub.yaml")
	subscriptionHandler := subscription.NewHandler(subscriptionSvc)

	svc.SetSubscriptionReferenceChecker(subscriptionSvc.ProviderReferencedByAnySubscription)

	apiMux := http.NewServeMux()
	handler.RegisterRoutes(apiMux)
	groupHandler.RegisterRoutes(apiMux)
	ruleHandler.RegisterRoutes(apiMux)
	outputHandler.RegisterRoutes(apiMux)
	subscriptionHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))

	scheduler := refresh.NewScheduler(repo, refreshSvc.RefreshProvider, time.Minute)
	go scheduler.Start(context.Background())

	log.Printf("[MAIN] Starting SubHub on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}
