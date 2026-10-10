package retention

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

const (
	wildcard              = "*"
	exactMatchSpecificity = math.MaxInt
)

func matchesPattern(pattern string, repository string) bool {
	expression := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), regexp.QuoteMeta(wildcard), ".*") + "$"
	return regexp.MustCompile(expression).MatchString(repository)
}

func specificity(pattern string) int {
	wildcardCount := strings.Count(pattern, wildcard)
	if wildcardCount == 0 {
		return exactMatchSpecificity
	}
	return len(pattern) - wildcardCount
}

func Resolve(policies []Policy, repository string) (*Policy, error) {
	var best *Policy
	bestPattern := ""
	bestSpecificity := -1
	for policyIndex := range policies {
		for _, pattern := range policies[policyIndex].Repositories {
			if !matchesPattern(pattern, repository) {
				continue
			}
			patternSpecificity := specificity(pattern)
			switch {
			case patternSpecificity > bestSpecificity:
				best, bestPattern, bestSpecificity = &policies[policyIndex], pattern, patternSpecificity
			case patternSpecificity == bestSpecificity && best != &policies[policyIndex]:
				return nil, fmt.Errorf("%w: %q and %q", ErrAmbiguousPolicy, bestPattern, pattern)
			}
		}
	}
	return best, nil
}
