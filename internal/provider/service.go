package provider

import (
	"context"
	"errors"

	"github.com/EricWvi/subhub/internal/config"
)

var ErrRefreshIntervalTooShort = errors.New("refresh interval must be at least 300 seconds")
var ErrNotFound = errors.New("provider not found")

type CreateProviderInput struct {
	Name                   string `json:"name"`
	URL                    string `json:"url"`
	RefreshIntervalSeconds int64  `json:"refresh_interval_seconds"`
}

type UpdateProviderInput struct {
	Name                   string `json:"name"`
	URL                    string `json:"url"`
	RefreshIntervalSeconds int64  `json:"refresh_interval_seconds"`
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateProviderInput) (Provider, error) {
	interval := in.RefreshIntervalSeconds
	if interval == 0 {
		interval = int64(config.DefaultRefreshInterval.Seconds())
	}
	if interval < 300 {
		return Provider{}, ErrRefreshIntervalTooShort
	}
	return s.repo.Create(ctx, Provider{
		Name:                   in.Name,
		URL:                    in.URL,
		RefreshIntervalSeconds: interval,
	})
}

func (s *Service) List(ctx context.Context) ([]Provider, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id int64) (Provider, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Provider{}, ErrNotFound
		}
		return Provider{}, err
	}
	return p, nil
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateProviderInput) (Provider, error) {
	interval := in.RefreshIntervalSeconds
	if interval == 0 {
		interval = int64(config.DefaultRefreshInterval.Seconds())
	}
	if interval < 300 {
		return Provider{}, ErrRefreshIntervalTooShort
	}
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Provider{}, ErrNotFound
		}
		return Provider{}, err
	}
	p.Name = in.Name
	p.URL = in.URL
	p.RefreshIntervalSeconds = interval
	return s.repo.Update(ctx, p)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
