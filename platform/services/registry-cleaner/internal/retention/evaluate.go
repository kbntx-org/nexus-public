package retention

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
)

func ValidatePolicies(policies []Policy) error {
	seenPatterns := map[string]bool{}
	for policyIndex, policy := range policies {
		for _, pattern := range policy.Repositories {
			if seenPatterns[pattern] {
				return fmt.Errorf("policies[%d]: %w: %s", policyIndex, ErrDuplicatePattern, pattern)
			}
			seenPatterns[pattern] = true
		}
		seenPriorities := map[int]bool{}
		for ruleIndex, rule := range policy.Rules {
			if seenPriorities[rule.Priority] {
				return fmt.Errorf("policies[%d].rules[%d]: %w: %d", policyIndex, ruleIndex, ErrDuplicatePriority, rule.Priority)
			}
			seenPriorities[rule.Priority] = true
			if err := validateRule(rule); err != nil {
				return fmt.Errorf("policies[%d].rules[%d]: %w", policyIndex, ruleIndex, err)
			}
		}
	}
	return nil
}

func validateRule(rule Rule) error {
	if rule.OlderThan.Duration < 0 {
		return ErrNegativeOlderThan
	}
	if rule.Newest > 0 && rule.OlderThan.Duration > 0 {
		return ErrConditionCount
	}
	return nil
}

func Evaluate(tags []Tag, rules []Rule, now time.Time) Plan {
	newestFirst := slices.Clone(tags)
	slices.SortFunc(newestFirst, func(left, right Tag) int {
		if byDate := right.PushedAt.Compare(left.PushedAt); byDate != 0 {
			return byDate
		}
		return strings.Compare(left.Name, right.Name)
	})

	byPriority := slices.Clone(rules)
	slices.SortFunc(byPriority, func(left, right Rule) int {
		return cmp.Compare(left.Priority, right.Priority)
	})

	ranks := make([]map[string]int, len(byPriority))
	for ruleIndex, rule := range byPriority {
		ranks[ruleIndex] = map[string]int{}
		for _, tag := range newestFirst {
			if rule.selects(tag) {
				ranks[ruleIndex][tag.Name] = len(ranks[ruleIndex])
			}
		}
	}

	var plan Plan
	for _, tag := range newestFirst {
		decision, action := decide(tag, byPriority, ranks, now)
		if action == ActionDelete {
			plan.Delete = append(plan.Delete, decision)
		} else {
			plan.Keep = append(plan.Keep, decision)
		}
	}
	return plan
}

func decide(tag Tag, rules []Rule, ranks []map[string]int, now time.Time) (Decision, Action) {
	if tag.PushedAt.IsZero() {
		return Decision{Tag: tag, Reason: ReasonUnknownPushDate}, ActionKeep
	}
	for ruleIndex := range rules {
		rule := &rules[ruleIndex]
		if rule.matches(tag, ranks[ruleIndex], now) {
			return Decision{Tag: tag, Reason: ReasonRule, Rule: rule}, rule.Action
		}
	}
	return Decision{Tag: tag, Reason: ReasonNoMatchingRule}, ActionKeep
}

func (r Rule) selects(tag Tag) bool {
	return r.TagPattern.Regexp == nil || r.TagPattern.MatchString(tag.Name)
}

func (r Rule) matches(tag Tag, ranks map[string]int, now time.Time) bool {
	if !r.selects(tag) {
		return false
	}
	switch {
	case r.Newest > 0:
		return ranks[tag.Name] < r.Newest
	case r.OlderThan.Duration > 0:
		return now.Sub(tag.PushedAt) > r.OlderThan.Duration
	default:
		return true
	}
}
