package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	mux.Handle("/", newFrontendHandler("dist"))

	scheduler := refresh.NewScheduler(repo, refreshSvc.RefreshProvider, time.Minute)
	go scheduler.Start(context.Background())

	log.Printf("[MAIN] Starting SubHub on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}

func newFrontendHandler(distDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(distDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}

		trimmedPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		targetPath := filepath.Join(distDir, filepath.FromSlash(trimmedPath))

		info, err := os.Stat(targetPath)
		if err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})
}
