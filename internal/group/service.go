package group

import (
	"context"
	"errors"
	"strings"
)

type CreateGroupInput struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}

type UpdateGroupInput struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

var ErrNameRequired = errors.New("name is required")

func (s *Service) Create(ctx context.Context, in CreateGroupInput) (ProxyGroup, error) {
	if strings.TrimSpace(in.Name) == "" {
		return ProxyGroup{}, ErrNameRequired
	}
	return s.repo.Create(ctx, ProxyGroup{
		Name:   strings.TrimSpace(in.Name),
		Script: in.Script,
	})
}

func (s *Service) GetByID(ctx context.Context, id int64) (ProxyGroup, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]ProxyGroup, error) {
	return s.repo.List(ctx)
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateGroupInput) (ProxyGroup, error) {
	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ProxyGroup{}, err
	}
	g.Name = strings.TrimSpace(in.Name)
	g.Script = strings.TrimSpace(in.Script)
	return s.repo.Update(ctx, g)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) ListNodes(ctx context.Context, groupID int64) ([]ProxyNodeView, error) {
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.repo.ListProxyNodeViews(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(group.Script) == "" {
		return nodes, nil
	}
	return s.selectNodes(group.Script, nodes)
}

func (s *Service) selectNodes(script string, nodes []ProxyNodeView) ([]ProxyNodeView, error) {
	selectedIDs, err := SelectNodeIDs(script, nodes)
	if err != nil {
		return nodes, nil
	}

	byID := map[int64]ProxyNodeView{}
	for _, node := range nodes {
		byID[node.ID] = node
	}

	var selected []ProxyNodeView
	for _, id := range selectedIDs {
		selected = append(selected, byID[id])
	}
	return selected, nil
}
