package retention

import (
	"errors"
	"testing"
)

func TestResolve(t *testing.T) {
	policies := []Policy{
		{Repositories: []string{"*"}},
		{Repositories: []string{"product/*"}},
		{Repositories: []string{"product/portfolio", "portfolio"}},
		{Repositories: []string{"platform-*"}},
	}
	testCases := []struct {
		repository   string
		expectPolicy int
	}{
		{repository: "smelt", expectPolicy: 0},
		{repository: "product/smelt", expectPolicy: 1},
		{repository: "product/portfolio", expectPolicy: 2},
		{repository: "portfolio", expectPolicy: 2},
		{repository: "platform-kiln", expectPolicy: 3},
		{repository: "product/nested/app", expectPolicy: 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.repository, func(t *testing.T) {
			policy, err := Resolve(policies, testCase.repository)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if policy != &policies[testCase.expectPolicy] {
				t.Errorf("resolved %v, expected %v", policy, policies[testCase.expectPolicy].Repositories)
			}
		})
	}
}

func TestResolveWithoutAMatch(t *testing.T) {
	policy, err := Resolve([]Policy{{Repositories: []string{"product/*"}}}, "smelt")
	if err != nil || policy != nil {
		t.Errorf("got %v (%v), expected no policy", policy, err)
	}
}

func TestResolveRejectsEquallySpecificPatterns(t *testing.T) {
	policies := []Policy{
		{Repositories: []string{"app-*"}},
		{Repositories: []string{"*-api"}},
	}
	if _, err := Resolve(policies, "app-api"); !errors.Is(err, ErrAmbiguousPolicy) {
		t.Errorf("got %v, expected %v", err, ErrAmbiguousPolicy)
	}
}

func TestResolveTreatsPatternSymbolsLiterally(t *testing.T) {
	policy, err := Resolve([]Policy{{Repositories: []string{"smelt.app"}}}, "smeltxapp")
	if err != nil || policy != nil {
		t.Errorf("got %v (%v), expected the dot to match only itself", policy, err)
	}
}
