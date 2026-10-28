package subscription

import (
	"context"
	"errors"
	"strings"

	customnode "github.com/EricWvi/subhub/internal/node"
	"github.com/EricWvi/subhub/internal/render"
)

var (
	ErrNodesRequired          = errors.New("at least one node is required")
	ErrDuplicateNodeSelection = errors.New("node can only be selected once per subscription")
)

func (s *Service) ListNodeProviders(ctx context.Context) ([]NodeProviderSubscription, error) {
	return s.repo.ListNodeProviders(ctx)
}

func (s *Service) GetNodeProviderByID(ctx context.Context, id int64) (NodeProviderSubscription, error) {
	return s.repo.GetNodeProviderByID(ctx, id)
}

func (s *Service) validateNodeProvider(ctx context.Context, in CreateNodeProviderSubscriptionInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return ErrSubscriptionNameRequired
	}
	if len(in.Nodes) == 0 {
		return ErrNodesRequired
	}
	seen := map[int64]bool{}
	for _, id := range in.Nodes {
		if id <= 0 {
			return ErrInvalidCustomNode
		}
		if seen[id] {
			return ErrDuplicateNodeSelection
		}
		seen[id] = true
		if _, err := s.nodeRepo.GetByID(ctx, id); err != nil {
			if errors.Is(err, customnode.ErrNotFound) {
				return ErrCustomNodeNotFound
			}
			return err
		}
	}
	return nil
}

func (s *Service) CreateNodeProvider(ctx context.Context, in CreateNodeProviderSubscriptionInput) (NodeProviderSubscription, error) {
	if err := s.validateNodeProvider(ctx, in); err != nil {
		return NodeProviderSubscription{}, err
	}
	in.Name = strings.TrimSpace(in.Name)
	return s.repo.CreateNodeProvider(ctx, in)
}

func (s *Service) UpdateNodeProvider(ctx context.Context, id int64, in UpdateNodeProviderSubscriptionInput) (NodeProviderSubscription, error) {
	if _, err := s.repo.GetNodeProviderByID(ctx, id); err != nil {
		return NodeProviderSubscription{}, err
	}
	if err := s.validateNodeProvider(ctx, in); err != nil {
		return NodeProviderSubscription{}, err
	}
	in.Name = strings.TrimSpace(in.Name)
	return s.repo.UpdateNodeProvider(ctx, id, in)
}

func (s *Service) DeleteNodeProvider(ctx context.Context, id int64) error {
	return s.repo.DeleteNodeProvider(ctx, id)
}

func (s *Service) BuildNodeProviderContent(ctx context.Context, id int64) (RenderedContent, error) {
	sub, err := s.repo.GetNodeProviderByID(ctx, id)
	if err != nil {
		return RenderedContent{}, err
	}
	nodes := make([]map[string]any, 0, len(sub.Nodes))
	for _, nodeID := range sub.Nodes {
		n, err := s.nodeRepo.GetByID(ctx, nodeID)
		if err != nil {
			if errors.Is(err, customnode.ErrNotFound) {
				continue
			}
			return RenderedContent{}, err
		}
		proxy, err := s.nodeSvc.ToProxy(n)
		if err != nil {
			return RenderedContent{}, err
		}
		nodes = append(nodes, proxy)
	}
	body, err := render.RenderProxyProviderSubscription(nodes)
	if err != nil {
		return RenderedContent{}, err
	}
	return RenderedContent{ContentType: "application/yaml", Body: body}, nil
}
