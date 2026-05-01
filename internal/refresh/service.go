package refresh

import (
	"context"
	"log"

	"github.com/EricWvi/subhub/internal/fetch"
	"github.com/EricWvi/subhub/internal/parse"
	"github.com/EricWvi/subhub/internal/provider"
)

type Service struct {
	providers *provider.Repository
	fetcher   *fetch.Client
}

func NewService(providers *provider.Repository, fetcher *fetch.Client) *Service {
	return &Service{providers: providers, fetcher: fetcher}
}

func (s *Service) RefreshProvider(ctx context.Context, providerID int64) error {
	log.Printf("[REFRESH] Starting refresh for provider %d", providerID)
	p, err := s.providers.GetByID(ctx, providerID)
	if err != nil {
		log.Printf("[REFRESH] Failed to get provider %d: %v", providerID, err)
		return err
	}

	log.Printf("[REFRESH] Fetching from %s", p.URL)
	payload, err := s.fetcher.Fetch(ctx, p.URL)
	if err != nil {
		log.Printf("[REFRESH] Fetch failed for provider %d: %v", providerID, err)
		_ = s.providers.RecordRefreshFailure(ctx, providerID, err.Error())
		return &provider.RefreshFailedError{Message: err.Error()}
	}

	log.Printf("[REFRESH] Parsing payload for provider %d", providerID)
	nodes, format, err := parse.DecodeAndNormalize(payload)
	if err != nil {
		log.Printf("[REFRESH] Parse failed for provider %d: %v", providerID, err)
		_ = s.providers.RecordRefreshFailure(ctx, providerID, err.Error())
		return &provider.RefreshFailedError{Message: err.Error()}
	}

	log.Printf("[REFRESH] Saving %d nodes for provider %d (format: %s)", len(nodes), providerID, format)
	err = s.providers.ReplaceLastKnownGoodSnapshot(ctx, providerID, format, nodes)
	if err != nil {
		log.Printf("[REFRESH] Save snapshot failed for provider %d: %v", providerID, err)
		return err
	}

	log.Printf("[REFRESH] Success for provider %d", providerID)
	return nil
}
