package node

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrNameRequired   = errors.New("name is required")
	ErrConfigRequired = errors.New("config is required")
	ErrInvalidConfig  = errors.New("config must be a valid YAML object")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateNodeInput) (Node, error) {
	name, config, err := normalizeAndValidate(in.Name, in.Config)
	if err != nil {
		return Node{}, err
	}
	return s.repo.Create(ctx, Node{Name: name, Config: config})
}

func (s *Service) List(ctx context.Context) ([]Node, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id int64) (Node, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateNodeInput) (Node, error) {
	name, config, err := normalizeAndValidate(in.Name, in.Config)
	if err != nil {
		return Node{}, err
	}
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Node{}, err
	}
	n.Name = name
	n.Config = config
	return s.repo.Update(ctx, n)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) ToProxy(n Node) (map[string]any, error) {
	var proxy map[string]any
	if err := yaml.Unmarshal([]byte(n.Config), &proxy); err != nil {
		return nil, fmt.Errorf("node %q: %w", n.Name, err)
	}
	if proxy == nil {
		return nil, fmt.Errorf("node %q: %w", n.Name, ErrInvalidConfig)
	}
	proxy["name"] = n.Name
	return proxy, nil
}

func normalizeAndValidate(name, config string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", ErrNameRequired
	}
	if strings.TrimSpace(config) == "" {
		return "", "", ErrConfigRequired
	}
	var proxy map[string]any
	if err := yaml.Unmarshal([]byte(config), &proxy); err != nil || proxy == nil {
		return "", "", ErrInvalidConfig
	}
	return name, config, nil
}
