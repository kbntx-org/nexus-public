package retention

import (
	"errors"
	"regexp"
	"slices"
	"testing"
	"time"
)

var now = time.Date(2026, time.October, 10, 12, 0, 0, 0, time.UTC)

func daysAgo(days int) time.Time {
	return now.Add(-time.Duration(days) * hoursPerDay * time.Hour)
}

func days(count int) Duration {
	return Duration{time.Duration(count) * hoursPerDay * time.Hour}
}

func pattern(expression string) Pattern {
	return Pattern{regexp.MustCompile(expression)}
}

func tagNames(decisions []Decision) []string {
	names := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		names = append(names, decision.Tag.Name)
	}
	slices.Sort(names)
	return names
}

func keptBy(plan Plan) map[string]int {
	priorities := map[string]int{}
	for _, decision := range plan.Keep {
		if decision.Rule != nil {
			priorities[decision.Tag.Name] = decision.Rule.Priority
		} else {
			priorities[decision.Tag.Name] = 0
		}
	}
	return priorities
}

func TestEvaluate(t *testing.T) {
	tags := []Tag{
		{Name: "latest", PushedAt: daysAgo(90)},
		{Name: "recent", PushedAt: daysAgo(1)},
		{Name: "second", PushedAt: daysAgo(2)},
		{Name: "third", PushedAt: daysAgo(20)},
		{Name: "fourth", PushedAt: daysAgo(30)},
		{Name: "v1.1.0", PushedAt: daysAgo(300)},
		{Name: "v1.0.0", PushedAt: daysAgo(400)},
		{Name: "unknown"},
	}

	testCases := []struct {
		name          string
		rules         []Rule
		expectDeleted []string
		expectKeptBy  map[string]int
	}{
		{
			name: "a higher priority keep protects tags a lower priority delete would take",
			rules: []Rule{
				{Priority: 1, Action: ActionKeep, TagPattern: pattern(`^latest$`)},
				{Priority: 2, Action: ActionKeep, Newest: 3},
				{Priority: 3, Action: ActionDelete, OlderThan: days(5)},
			},
			expectDeleted: []string{"fourth", "v1.0.0", "v1.1.0"},
			expectKeptBy:  map[string]int{"latest": 1, "recent": 2, "second": 2, "third": 2, "unknown": 0},
		},
		{
			name: "rules run by priority, not by the order they are listed in",
			rules: []Rule{
				{Priority: 2, Action: ActionKeep, Newest: 3},
				{Priority: 1, Action: ActionDelete, OlderThan: days(5)},
			},
			expectDeleted: []string{"fourth", "latest", "third", "v1.0.0", "v1.1.0"},
			expectKeptBy:  map[string]int{"recent": 2, "second": 2, "unknown": 0},
		},
		{
			name: "newest counts only the tags the rule selects",
			rules: []Rule{
				{Priority: 1, Action: ActionKeep, TagPattern: pattern(`^v\d`), Newest: 1},
				{Priority: 2, Action: ActionDelete, TagPattern: pattern(`^v\d`)},
			},
			expectDeleted: []string{"v1.0.0"},
			expectKeptBy:  map[string]int{"v1.1.0": 1, "latest": 0, "recent": 0, "second": 0, "third": 0, "fourth": 0, "unknown": 0},
		},
		{
			name: "a catch-all delete takes every tag no earlier rule kept",
			rules: []Rule{
				{Priority: 1, Action: ActionKeep, Newest: 1},
				{Priority: 2, Action: ActionDelete},
			},
			expectDeleted: []string{"fourth", "latest", "second", "third", "v1.0.0", "v1.1.0"},
			expectKeptBy:  map[string]int{"recent": 1, "unknown": 0},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			plan := Evaluate(tags, testCase.rules, now)
			if deleted := tagNames(plan.Delete); !slices.Equal(deleted, testCase.expectDeleted) {
				t.Errorf("deleted %v, expected %v", deleted, testCase.expectDeleted)
			}
			kept := keptBy(plan)
			if len(kept) != len(testCase.expectKeptBy) {
				t.Errorf("kept %v, expected %v", kept, testCase.expectKeptBy)
			}
			for name, expected := range testCase.expectKeptBy {
				if priority, found := kept[name]; !found || priority != expected {
					t.Errorf("tag %s kept by rule %d (found %t), expected rule %d", name, priority, found, expected)
				}
			}
		})
	}
}

func TestEvaluateKeepsUnknownPushDatesEvenFromACatchAllDelete(t *testing.T) {
	plan := Evaluate([]Tag{{Name: "unknown"}}, []Rule{{Priority: 1, Action: ActionDelete}}, now)
	if len(plan.Delete) != 0 || plan.Keep[0].Reason != ReasonUnknownPushDate {
		t.Errorf("unexpected plan %+v", plan)
	}
}

func TestEvaluateBreaksPushDateTiesByName(t *testing.T) {
	tags := []Tag{
		{Name: "beta", PushedAt: daysAgo(10)},
		{Name: "alpha", PushedAt: daysAgo(10)},
	}
	rules := []Rule{
		{Priority: 1, Action: ActionKeep, Newest: 1},
		{Priority: 2, Action: ActionDelete},
	}
	plan := Evaluate(tags, rules, now)
	if deleted := tagNames(plan.Delete); !slices.Equal(deleted, []string{"beta"}) {
		t.Errorf("deleted %v, expected [beta]", deleted)
	}
}

func TestValidatePolicies(t *testing.T) {
	validRule := Rule{Priority: 1, Action: ActionKeep, Newest: 1}
	testCases := []struct {
		name     string
		policies []Policy
		expected error
	}{
		{name: "valid", policies: []Policy{{Repositories: []string{"smelt"}, Rules: []Rule{validRule}}}},
		{
			name: "duplicate pattern",
			policies: []Policy{
				{Repositories: []string{"*"}, Rules: []Rule{validRule}},
				{Repositories: []string{"*"}, Rules: []Rule{validRule}},
			},
			expected: ErrDuplicatePattern,
		},
		{
			name:     "duplicate priority",
			policies: []Policy{{Repositories: []string{"smelt"}, Rules: []Rule{validRule, validRule}}},
			expected: ErrDuplicatePriority,
		},
		{
			name: "rule with two conditions",
			policies: []Policy{{Repositories: []string{"smelt"}, Rules: []Rule{
				{Priority: 1, Action: ActionDelete, Newest: 1, OlderThan: days(1)},
			}}},
			expected: ErrConditionCount,
		},
		{
			name: "negative olderThan",
			policies: []Policy{{Repositories: []string{"smelt"}, Rules: []Rule{
				{Priority: 1, Action: ActionDelete, OlderThan: Duration{-time.Hour}},
			}}},
			expected: ErrNegativeOlderThan,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidatePolicies(testCase.policies)
			if !errors.Is(err, testCase.expected) {
				t.Errorf("got %v, expected %v", err, testCase.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	testCases := []struct {
		value     string
		expected  time.Duration
		expectErr bool
	}{
		{value: "5d", expected: 5 * hoursPerDay * time.Hour},
		{value: "36h", expected: 36 * time.Hour},
		{value: "90m", expected: 90 * time.Minute},
		{value: "xd", expectErr: true},
		{value: "5", expectErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.value, func(t *testing.T) {
			parsed, err := ParseDuration(testCase.value)
			if testCase.expectErr {
				if err == nil {
					t.Errorf("expected an error, got %v", parsed)
				}
				return
			}
			if err != nil || parsed.Duration != testCase.expected {
				t.Errorf("got %v (%v), expected %v", parsed.Duration, err, testCase.expected)
			}
		})
	}
}
