package rule

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrRuleTypeRequired   = errors.New("rule type is required")
	ErrPatternRequired    = errors.New("pattern is required")
	ErrProxyGroupRequired = errors.New("proxy group is required")
	ErrInvalidProxyGroup  = errors.New("invalid proxy group")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) resolveTarget(ctx context.Context, proxyGroup string) (string, sql.NullInt64, error) {
	switch strings.TrimSpace(proxyGroup) {
	case "":
		return "", sql.NullInt64{}, ErrProxyGroupRequired
	case "DIRECT", "REJECT":
		return strings.TrimSpace(proxyGroup), sql.NullInt64{}, nil
	default:
		id, err := s.repo.FindProxyGroupIDByName(ctx, strings.TrimSpace(proxyGroup))
		if err != nil {
			return "", sql.NullInt64{}, ErrInvalidProxyGroup
		}
		return "PROXY_GROUP", sql.NullInt64{Int64: id, Valid: true}, nil
	}
}

func (s *Service) Create(ctx context.Context, in CreateRuleInput) (Rule, error) {
	if strings.TrimSpace(in.RuleType) == "" {
		return Rule{}, ErrRuleTypeRequired
	}
	if strings.TrimSpace(in.Pattern) == "" {
		return Rule{}, ErrPatternRequired
	}

	targetKind, proxyGroupID, err := s.resolveTarget(ctx, in.ProxyGroup)
	if err != nil {
		return Rule{}, err
	}

	return s.repo.Create(ctx, CreateRuleRecord{
		RuleType:    strings.TrimSpace(in.RuleType),
		Pattern:     strings.TrimSpace(in.Pattern),
		TargetKind:  targetKind,
		ProxyGroupID: proxyGroupID,
	})
}

func (s *Service) GetByID(ctx context.Context, id int64) (Rule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateRuleInput) (Rule, error) {
	if strings.TrimSpace(in.RuleType) == "" {
		return Rule{}, ErrRuleTypeRequired
	}
	if strings.TrimSpace(in.Pattern) == "" {
		return Rule{}, ErrPatternRequired
	}

	targetKind, proxyGroupID, err := s.resolveTarget(ctx, in.ProxyGroup)
	if err != nil {
		return Rule{}, err
	}

	return s.repo.Update(ctx, id, CreateRuleRecord{
		RuleType:    strings.TrimSpace(in.RuleType),
		Pattern:     strings.TrimSpace(in.Pattern),
		TargetKind:  targetKind,
		ProxyGroupID: proxyGroupID,
	})
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Import(ctx context.Context, in ImportRulesInput) (*ImportRulesResult, error) {
	lines := strings.Split(strings.TrimSpace(in.Rules), "\n")
	if in.Reverse {
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
	}

	var records []CreateRuleRecord
	skipped := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		parts := strings.SplitN(line, ",", 3)
		if len(parts) != 3 {
			skipped++
			continue
		}

		ruleType := strings.TrimSpace(parts[0])
		pattern := strings.TrimSpace(parts[1])
		target := strings.TrimSpace(parts[2])
		if ruleType == "" || pattern == "" || target == "" {
			skipped++
			continue
		}

		targetKind, proxyGroupID, err := s.resolveTarget(ctx, target)
		if err != nil {
			skipped++
			continue
		}

		records = append(records, CreateRuleRecord{
			RuleType:     ruleType,
			Pattern:      pattern,
			TargetKind:   targetKind,
			ProxyGroupID: proxyGroupID,
		})
	}

	if len(records) == 0 {
		return &ImportRulesResult{Imported: 0, Skipped: skipped}, nil
	}

	imported, err := s.repo.BatchImport(ctx, records)
	if err != nil {
		return nil, err
	}
	return &ImportRulesResult{Imported: imported, Skipped: skipped}, nil
}

func (s *Service) List(ctx context.Context, in ListRulesInput) (*ListRulesResult, error) {
	in = normalizeListInput(in)
	rules, total, err := s.repo.List(ctx, in)
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []Rule{}
	}
	return &ListRulesResult{
		Rules:    rules,
		Page:     in.Page,
		PageSize: in.PageSize,
		Total:    total,
	}, nil
}
