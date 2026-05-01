package refresh

import (
	"context"

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
	p, err := s.providers.GetByID(ctx, providerID)
	if err != nil {
		return err
	}

	payload, err := s.fetcher.Fetch(ctx, p.URL)
	if err != nil {
		_ = s.providers.RecordRefreshFailure(ctx, providerID, err.Error())
		return &provider.RefreshFailedError{Message: err.Error()}
	}

	nodes, format, err := parse.DecodeAndNormalize(payload)
	if err != nil {
		_ = s.providers.RecordRefreshFailure(ctx, providerID, err.Error())
		return &provider.RefreshFailedError{Message: err.Error()}
	}

	return s.providers.ReplaceLastKnownGoodSnapshot(ctx, providerID, format, payload, nodes)
}
