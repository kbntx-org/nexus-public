package retention

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type Registry interface {
	ListRepositories(ctx context.Context) ([]string, error)
	ListTags(ctx context.Context, repository string) ([]Tag, error)
	DeleteTag(ctx context.Context, repository string, tag string) error
}

type Service struct {
	registry Registry
	policies []Policy
	dryRun   bool
	now      func() time.Time
}

func NewService(registry Registry, policies []Policy, dryRun bool, now func() time.Time) *Service {
	return &Service{
		registry: registry,
		policies: policies,
		dryRun:   dryRun,
		now:      now,
	}
}

func (s *Service) Run(ctx context.Context) error {
	repositories, err := s.registry.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("list repositories: %w", err)
	}

	var failures []error
	for _, repository := range repositories {
		policy, err := Resolve(s.policies, repository)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", repository, err))
			continue
		}
		if policy == nil {
			slog.Info("no policy covers repository", "repository", repository)
			continue
		}
		if err := s.cleanRepository(ctx, repository, policy.Rules); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", repository, err))
		}
	}
	return errors.Join(failures...)
}

func (s *Service) cleanRepository(ctx context.Context, repository string, rules []Rule) error {
	tags, err := s.registry.ListTags(ctx, repository)
	if err != nil {
		return fmt.Errorf("list tags: %w", err)
	}
	plan := Evaluate(tags, rules, s.now())

	for _, kept := range plan.Keep {
		slog.Debug("keeping tag", decisionAttributes(repository, kept)...)
	}

	var failures []error
	for _, deleted := range plan.Delete {
		if s.dryRun {
			slog.Info("would delete tag", decisionAttributes(repository, deleted)...)
			continue
		}
		if err := s.registry.DeleteTag(ctx, repository, deleted.Tag.Name); err != nil {
			failures = append(failures, fmt.Errorf("delete tag %s: %w", deleted.Tag.Name, err))
			continue
		}
		slog.Info("deleted tag", decisionAttributes(repository, deleted)...)
	}

	slog.Info("repository cleaned",
		"repository", repository,
		"kept", len(plan.Keep),
		"toDelete", len(plan.Delete),
		"failed", len(failures),
		"dryRun", s.dryRun,
	)
	return errors.Join(failures...)
}

func decisionAttributes(repository string, decision Decision) []any {
	attributes := []any{
		"repository", repository,
		"tag", decision.Tag.Name,
		"pushedAt", decision.Tag.PushedAt,
		"reason", decision.Reason,
	}
	if decision.Rule != nil {
		attributes = append(attributes, "rulePriority", decision.Rule.Priority, "ruleDescription", decision.Rule.Description)
	}
	return attributes
}
