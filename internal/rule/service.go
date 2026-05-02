package rule

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, page, pageSize int) (*ListRulesResult, error) {
	rules, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []Rule{}
	}
	return &ListRulesResult{
		Rules:    rules,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}
