package refresh

import (
	"context"
	"log"
	"time"

	"github.com/EricWvi/subhub/internal/provider"
)

type ProviderLister interface {
	List(ctx context.Context) ([]provider.Provider, error)
}

type RefreshFunc func(ctx context.Context, providerID int64) error

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type Scheduler struct {
	repo     ProviderLister
	refresh  RefreshFunc
	interval time.Duration
	logger   *log.Logger
	clock    Clock
}

func NewScheduler(repo ProviderLister, refresh RefreshFunc, interval time.Duration) *Scheduler {
	return &Scheduler{
		repo:     repo,
		refresh:  refresh,
		interval: interval,
		logger:   log.Default(),
		clock:    realClock{},
	}
}

func (s *Scheduler) WithClock(c Clock) *Scheduler {
	s.clock = c
	return s
}

func (s *Scheduler) WithLogger(l *log.Logger) *Scheduler {
	s.logger = l
	return s
}

func (s *Scheduler) RunOnce(ctx context.Context) {
	providers, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Printf("scheduler: list providers: %v", err)
		return
	}

	now := s.clock.Now().UTC()
	for _, p := range providers {
		if now.Sub(p.UpdatedAt) < time.Duration(p.RefreshIntervalSeconds)*time.Second {
			continue
		}
		if err := s.refresh(ctx, p.ID); err != nil {
			s.logger.Printf("scheduler: refresh provider %d: %v", p.ID, err)
		}
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.RunOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RunOnce(ctx)
		}
	}
}
