package retention

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
)

type fakeRegistry struct {
	tags         map[string][]Tag
	listFailures map[string]error
	deleted      map[string][]string
}

func (f *fakeRegistry) ListRepositories(context.Context) ([]string, error) {
	return slices.Sorted(maps.Keys(f.tags)), nil
}

func (f *fakeRegistry) ListTags(_ context.Context, repository string) ([]Tag, error) {
	if err := f.listFailures[repository]; err != nil {
		return nil, err
	}
	return f.tags[repository], nil
}

func (f *fakeRegistry) DeleteTag(_ context.Context, repository string, tag string) error {
	f.deleted[repository] = append(f.deleted[repository], tag)
	return nil
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{
		tags: map[string][]Tag{
			"smelt": {
				{Name: "new", PushedAt: daysAgo(1)},
				{Name: "old", PushedAt: daysAgo(40)},
			},
			"kiln": {
				{Name: "new", PushedAt: daysAgo(1)},
				{Name: "old", PushedAt: daysAgo(40)},
			},
		},
		listFailures: map[string]error{},
		deleted:      map[string][]string{},
	}
}

var deleteOlderThanOneWeek = []Policy{{
	Repositories: []string{"smelt", "kiln"},
	Rules:        []Rule{{Priority: 1, Action: ActionDelete, OlderThan: days(7)}},
}}

func clock() time.Time { return now }

func TestServiceDeletesTagsADeleteRuleMatches(t *testing.T) {
	registry := newFakeRegistry()
	if err := NewService(registry, deleteOlderThanOneWeek, false, clock).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, repository := range []string{"smelt", "kiln"} {
		if !slices.Equal(registry.deleted[repository], []string{"old"}) {
			t.Errorf("%s deleted %v, expected [old]", repository, registry.deleted[repository])
		}
	}
}

func TestServiceDryRunDeletesNothing(t *testing.T) {
	registry := newFakeRegistry()
	if err := NewService(registry, deleteOlderThanOneWeek, true, clock).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(registry.deleted) != 0 {
		t.Errorf("dry run deleted %v", registry.deleted)
	}
}

func TestServiceContinuesAfterARepositoryFails(t *testing.T) {
	registry := newFakeRegistry()
	listFailure := errors.New("not found")
	registry.listFailures["smelt"] = listFailure

	err := NewService(registry, deleteOlderThanOneWeek, false, clock).Run(context.Background())
	if !errors.Is(err, listFailure) {
		t.Errorf("got %v, expected the smelt failure", err)
	}
	if !slices.Equal(registry.deleted["kiln"], []string{"old"}) {
		t.Errorf("kiln deleted %v, expected [old]", registry.deleted["kiln"])
	}
}

func TestServiceAppliesTheMostSpecificPolicy(t *testing.T) {
	registry := newFakeRegistry()
	registry.tags["portfolio"] = []Tag{{Name: "old", PushedAt: daysAgo(40)}}
	policies := []Policy{
		{Repositories: []string{"*"}, Rules: []Rule{{Priority: 1, Action: ActionDelete, OlderThan: days(7)}}},
		{Repositories: []string{"kiln"}, Rules: []Rule{{Priority: 1, Action: ActionKeep}}},
	}

	if err := NewService(registry, policies, false, clock).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(registry.deleted["kiln"]) != 0 {
		t.Errorf("kiln deleted %v, expected its own policy to keep everything", registry.deleted["kiln"])
	}
	for _, repository := range []string{"smelt", "portfolio"} {
		if !slices.Equal(registry.deleted[repository], []string{"old"}) {
			t.Errorf("%s deleted %v, expected the wildcard policy to delete [old]", repository, registry.deleted[repository])
		}
	}
}

func TestServiceLeavesRepositoriesNoPolicyCovers(t *testing.T) {
	registry := newFakeRegistry()
	policies := []Policy{{Repositories: []string{"smelt"}, Rules: []Rule{{Priority: 1, Action: ActionDelete}}}}

	if err := NewService(registry, policies, false, clock).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(registry.deleted["kiln"]) != 0 {
		t.Errorf("kiln deleted %v without a policy", registry.deleted["kiln"])
	}
}
