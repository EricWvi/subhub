package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/EricWvi/subhub/internal/config"
)

var ErrRefreshIntervalTooShort = errors.New("refresh interval must be at least 5 minutes")
var ErrNotFound = errors.New("provider not found")
var ErrInvalidURL = errors.New("invalid provider url")
var errInvalidAbbrev = errors.New("abbrev must contain uppercase letters only")

func normalizeAbbrev(raw string) (string, error) {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	if upper == "" {
		return "", nil
	}
	for _, r := range upper {
		if r < 'A' || r > 'Z' {
			return "", errInvalidAbbrev
		}
	}
	return upper, nil
}

type CreateProviderInput struct {
	Name                   string `json:"name"`
	URL                    string `json:"url"`
	RefreshIntervalMinutes int64  `json:"refresh_interval_minutes"`
	Abbrev                 string `json:"abbrev"`
}

type UpdateProviderInput struct {
	Name                   string `json:"name"`
	URL                    string `json:"url"`
	RefreshIntervalMinutes int64  `json:"refresh_interval_minutes"`
	Abbrev                 string `json:"abbrev"`
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateProviderInput) (Provider, error) {
	if _, err := url.ParseRequestURI(in.URL); err != nil {
		return Provider{}, fmt.Errorf("%w: %s", ErrInvalidURL, err.Error())
	}
	abbrev, err := normalizeAbbrev(in.Abbrev)
	if err != nil {
		return Provider{}, err
	}
	interval := in.RefreshIntervalMinutes
	if interval == 0 {
		interval = int64(config.DefaultRefreshInterval.Minutes())
	}
	if interval < 5 {
		return Provider{}, ErrRefreshIntervalTooShort
	}
	return s.repo.Create(ctx, Provider{
		Name:                   in.Name,
		URL:                    in.URL,
		RefreshIntervalMinutes: interval,
		Abbrev:                 abbrev,
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
	if _, err := url.ParseRequestURI(in.URL); err != nil {
		return Provider{}, fmt.Errorf("%w: %s", ErrInvalidURL, err.Error())
	}
	abbrev, err := normalizeAbbrev(in.Abbrev)
	if err != nil {
		return Provider{}, err
	}
	interval := in.RefreshIntervalMinutes
	if interval == 0 {
		interval = int64(config.DefaultRefreshInterval.Minutes())
	}
	if interval < 5 {
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
	p.RefreshIntervalMinutes = interval
	p.Abbrev = abbrev
	return s.repo.Update(ctx, p)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetLatestSnapshot(ctx context.Context, id int64) (Snapshot, error) {
	return s.repo.GetLatestSnapshot(ctx, id)
}
