package group

import (
	"context"
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
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
var ErrDeleteReferenced = errors.New("proxy group is referenced by other resources")

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
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return ErrDeleteReferenced
	}
	return nil
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

func (s *Service) ResolveNodesForOutput(ctx context.Context, groupID int64, allowedProviderIDs []int64) ([]string, []map[string]any, error) {
	g, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}

	rawNodes, err := s.repo.ListRawNodesByProviders(ctx, allowedProviderIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(rawNodes) == 0 {
		return nil, nil, nil
	}

	var filteredNodes []ResolvedNode
	if strings.TrimSpace(g.Script) != "" {
		var views []ProxyNodeView
		for _, n := range rawNodes {
			views = append(views, ProxyNodeView{ID: n.ID, Name: n.Name})
		}
		selectedIDs, err := SelectNodeIDs(g.Script, views)
		if err != nil {
			filteredNodes = rawNodes
		} else {
			idSet := map[int64]bool{}
			for _, id := range selectedIDs {
				idSet[id] = true
			}
			for _, n := range rawNodes {
				if idSet[n.ID] {
					filteredNodes = append(filteredNodes, n)
				}
			}
		}
	} else {
		filteredNodes = rawNodes
	}

	var names []string
	var nodes []map[string]any
	for _, n := range filteredNodes {
		var node map[string]any
		if err := yaml.Unmarshal([]byte(n.RawYAML), &node); err != nil {
			continue
		}
		names = append(names, n.Name)
		nodes = append(nodes, node)
	}
	return names, nodes, nil
}
