package subscription

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/EricWvi/subhub/internal/group"
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/render"
	"github.com/EricWvi/subhub/internal/rule"
)

var (
	ErrSubscriptionNameRequired = errors.New("subscription name is required")
	ErrProvidersRequired        = errors.New("at least one provider is required")
	ErrReservedProxyGroup       = errors.New("Proxies proxy-group cannot be deleted")
	ErrDuplicateRuleBinding     = errors.New("internal proxy group already bound")
	ErrNotFound                 = errors.New("subscription not found")
)

type RenderedContent struct {
	ContentType          string
	SubscriptionUserinfo string
	Body                 string
}

type Service struct {
	repo         *Repository
	providerRepo *provider.Repository
	groupSvc     *group.Service
	ruleRepo     *rule.Repository
	templatePath string
}

func NewService(repo *Repository, providerRepo *provider.Repository, groupSvc *group.Service, ruleRepo *rule.Repository, templatePath string) *Service {
	return &Service{
		repo:         repo,
		providerRepo: providerRepo,
		groupSvc:     groupSvc,
		ruleRepo:     ruleRepo,
		templatePath: templatePath,
	}
}

func (s *Service) ListClashConfigs(ctx context.Context) ([]ClashConfigSubscription, error) {
	return s.repo.ListClashConfigs(ctx)
}

func (s *Service) ListProxyProviders(ctx context.Context) ([]ProxyProviderSubscription, error) {
	return s.repo.ListProxyProviders(ctx)
}

func (s *Service) ListRuleProviders(ctx context.Context) ([]RuleProviderSubscription, error) {
	return s.repo.ListRuleProviders(ctx)
}

func (s *Service) CreateClashConfig(ctx context.Context, in CreateClashConfigSubscriptionInput) (ClashConfigSubscription, error) {
	if in.Name == "" {
		return ClashConfigSubscription{}, ErrSubscriptionNameRequired
	}
	if len(in.Providers) == 0 {
		return ClashConfigSubscription{}, ErrProvidersRequired
	}

	sub, err := s.repo.CreateClashConfig(ctx, in)
	if err != nil {
		return ClashConfigSubscription{}, err
	}

	sysPG, err := s.repo.CreateSystemProxyGroup(ctx, sub.ID, CreateClashConfigProxyGroupInput{
		Name:    "Proxies",
		Type:    "select",
		Proxies: []ProxyMember{{Type: "DIRECT", Value: "DIRECT"}},
	})
	if err != nil {
		return ClashConfigSubscription{}, err
	}

	sub.ProxyGroups = append([]ClashConfigProxyGroup{sysPG}, sub.ProxyGroups...)
	return sub, nil
}

func (s *Service) GetClashConfigByID(ctx context.Context, id int64) (ClashConfigSubscription, error) {
	return s.repo.GetClashConfigByID(ctx, id)
}

func (s *Service) UpdateClashConfig(ctx context.Context, id int64, in UpdateClashConfigSubscriptionInput) (ClashConfigSubscription, error) {
	if in.Name == "" {
		return ClashConfigSubscription{}, ErrSubscriptionNameRequired
	}
	if len(in.Providers) == 0 {
		return ClashConfigSubscription{}, ErrProvidersRequired
	}
	return s.repo.UpdateClashConfig(ctx, id, in)
}

func (s *Service) DeleteClashConfig(ctx context.Context, id int64) error {
	return s.repo.DeleteClashConfig(ctx, id)
}

func (s *Service) CreateProxyProvider(ctx context.Context, in CreateProxyProviderSubscriptionInput) (ProxyProviderSubscription, error) {
	if in.Name == "" {
		return ProxyProviderSubscription{}, ErrSubscriptionNameRequired
	}
	if len(in.Providers) == 0 {
		return ProxyProviderSubscription{}, ErrProvidersRequired
	}
	return s.repo.CreateProxyProvider(ctx, in)
}

func (s *Service) GetProxyProviderByID(ctx context.Context, id int64) (ProxyProviderSubscription, error) {
	return s.repo.GetProxyProviderByID(ctx, id)
}

func (s *Service) DeleteProxyProvider(ctx context.Context, id int64) error {
	return s.repo.DeleteProxyProvider(ctx, id)
}

func (s *Service) UpdateProxyProvider(ctx context.Context, id int64, in UpdateProxyProviderSubscriptionInput) (ProxyProviderSubscription, error) {
	if in.Name == "" {
		return ProxyProviderSubscription{}, ErrSubscriptionNameRequired
	}
	if len(in.Providers) == 0 {
		return ProxyProviderSubscription{}, ErrProvidersRequired
	}
	return s.repo.UpdateProxyProvider(ctx, id, in)
}

func (s *Service) CreateRuleProvider(ctx context.Context, in CreateRuleProviderSubscriptionInput) (RuleProviderSubscription, error) {
	if in.Name == "" {
		return RuleProviderSubscription{}, ErrSubscriptionNameRequired
	}
	if len(in.Providers) == 0 {
		return RuleProviderSubscription{}, ErrProvidersRequired
	}
	return s.repo.CreateRuleProvider(ctx, in)
}

func (s *Service) GetRuleProviderByID(ctx context.Context, id int64) (RuleProviderSubscription, error) {
	return s.repo.GetRuleProviderByID(ctx, id)
}

func (s *Service) DeleteRuleProvider(ctx context.Context, id int64) error {
	return s.repo.DeleteRuleProvider(ctx, id)
}

func (s *Service) UpdateRuleProvider(ctx context.Context, id int64, in UpdateRuleProviderSubscriptionInput) (RuleProviderSubscription, error) {
	if in.Name == "" {
		return RuleProviderSubscription{}, ErrSubscriptionNameRequired
	}
	if len(in.Providers) == 0 {
		return RuleProviderSubscription{}, ErrProvidersRequired
	}
	return s.repo.UpdateRuleProvider(ctx, id, in)
}

func (s *Service) ProviderReferencedByAnySubscription(ctx context.Context, providerID int64) (bool, error) {
	return s.repo.ProviderReferencedByAnySubscription(ctx, providerID)
}

func (s *Service) BuildClashConfigContent(ctx context.Context, id int64) (RenderedContent, error) {
	sub, err := s.repo.GetClashConfigByID(ctx, id)
	if err != nil {
		return RenderedContent{}, err
	}

	var validProviders []provider.Provider
	for _, pid := range sub.Providers {
		p, err := s.providerRepo.GetByID(ctx, pid)
		if err != nil {
			continue
		}
		if p.Total > 0 && float64(p.Used)/float64(p.Total) < 0.99 {
			validProviders = append(validProviders, p)
		}
	}

	var allowedProviderIDs []int64
	for _, p := range validProviders {
		allowedProviderIDs = append(allowedProviderIDs, p.ID)
	}

	var allNodes []map[string]any
	seen := map[string]bool{}
	for _, pg := range sub.ProxyGroups {
		if pg.BindInternalGroupID != 0 {
			_, nodes, err := s.groupSvc.ResolveNodesForOutput(ctx, pg.BindInternalGroupID, allowedProviderIDs)
			if err != nil {
				continue
			}
			for _, n := range nodes {
				name, _ := n["name"].(string)
				if !seen[name] {
					seen[name] = true
					allNodes = append(allNodes, n)
				}
			}
		}
	}

	bindMap := map[int64]string{}
	for _, pg := range sub.ProxyGroups {
		if pg.BindInternalGroupID != 0 {
			bindMap[pg.BindInternalGroupID] = pg.Name
		}
	}

	var renderedGroups []render.RenderedProxyGroup
	for _, pg := range sub.ProxyGroups {
		rp := render.RenderedProxyGroup{
			Name:     pg.Name,
			Type:     pg.Type,
			URL:      pg.URL,
			Interval: pg.Interval,
		}
		for _, m := range pg.Proxies {
			switch m.Type {
			case "internal":
				groupID, err := strconv.ParseInt(m.Value, 10, 64)
				if err != nil {
					continue
				}
				names, _, err := s.groupSvc.ResolveNodesForOutput(ctx, groupID, allowedProviderIDs)
				if err != nil {
					continue
				}
				rp.Proxies = append(rp.Proxies, names...)
			default:
				rp.Proxies = append(rp.Proxies, m.Value)
			}
		}
		renderedGroups = append(renderedGroups, rp)
	}

	ruleRows, err := s.ruleRepo.ListAscendingForOutput(ctx)
	if err != nil {
		return RenderedContent{}, err
	}

	var mappedRules []string
	for _, r := range ruleRows {
		var target string
		if r.TargetKind == "PROXY_GROUP" && r.ProxyGroupID.Valid {
			if name, ok := bindMap[r.ProxyGroupID.Int64]; ok {
				target = name
			} else if r.ProxyGroupName.Valid {
				target = r.ProxyGroupName.String
			} else {
				continue
			}
		} else {
			target = r.TargetKind
		}
		mappedRules = append(mappedRules, r.RuleType+","+r.Pattern+","+target)
	}

	yamlText, err := render.RenderClashConfigSubscription(s.templatePath, allNodes, renderedGroups, mappedRules)
	if err != nil {
		return RenderedContent{}, err
	}

	var userinfo string
	if len(validProviders) > 0 {
		p := validProviders[0]
		userinfo = fmt.Sprintf("upload=0; download=%d; total=%d; expire=%d", p.Used, p.Total, p.Expire)
	}

	return RenderedContent{
		ContentType:          "application/yaml",
		SubscriptionUserinfo: userinfo,
		Body:                 yamlText,
	}, nil
}

func (s *Service) BuildProxyProviderContent(ctx context.Context, id int64) (RenderedContent, error) {
	sub, err := s.repo.GetProxyProviderByID(ctx, id)
	if err != nil {
		return RenderedContent{}, err
	}

	var validProviders []provider.Provider
	for _, pid := range sub.Providers {
		p, err := s.providerRepo.GetByID(ctx, pid)
		if err != nil {
			continue
		}
		if p.Total > 0 && float64(p.Used)/float64(p.Total) < 0.99 {
			validProviders = append(validProviders, p)
		}
	}

	var allowedProviderIDs []int64
	for _, p := range validProviders {
		allowedProviderIDs = append(allowedProviderIDs, p.ID)
	}

	_, nodes, err := s.groupSvc.ResolveNodesForOutput(ctx, sub.InternalProxyGroupID, allowedProviderIDs)
	if err != nil {
		return RenderedContent{}, err
	}

	body, err := render.RenderProxyProviderSubscription(nodes)
	if err != nil {
		return RenderedContent{}, err
	}

	var userinfo string
	if len(validProviders) > 0 {
		p := validProviders[0]
		userinfo = fmt.Sprintf("upload=0; download=%d; total=%d; expire=%d", p.Used, p.Total, p.Expire)
	}

	return RenderedContent{
		ContentType:          "application/yaml",
		SubscriptionUserinfo: userinfo,
		Body:                 body,
	}, nil
}

func (s *Service) BuildRuleProviderContent(ctx context.Context, id int64) (RenderedContent, error) {
	sub, err := s.repo.GetRuleProviderByID(ctx, id)
	if err != nil {
		return RenderedContent{}, err
	}

	var validProviders []provider.Provider
	for _, pid := range sub.Providers {
		p, err := s.providerRepo.GetByID(ctx, pid)
		if err != nil {
			continue
		}
		if p.Total > 0 && float64(p.Used)/float64(p.Total) < 0.99 {
			validProviders = append(validProviders, p)
		}
	}

	rules, err := s.ruleRepo.ListForInternalGroup(ctx, sub.InternalProxyGroupID)
	if err != nil {
		return RenderedContent{}, err
	}

	body, err := render.RenderRuleProviderSubscription(rules)
	if err != nil {
		return RenderedContent{}, err
	}

	var userinfo string
	if len(validProviders) > 0 {
		p := validProviders[0]
		userinfo = fmt.Sprintf("upload=0; download=%d; total=%d; expire=%d", p.Used, p.Total, p.Expire)
	}

	return RenderedContent{
		ContentType:          "application/yaml",
		SubscriptionUserinfo: userinfo,
		Body:                 body,
	}, nil
}
