package retention

import "errors"

var (
	ErrDuplicatePattern  = errors.New("repository pattern is listed more than once")
	ErrAmbiguousPolicy   = errors.New("two equally specific patterns match the repository")
	ErrDuplicatePriority = errors.New("two rules share the same priority")
	ErrConditionCount    = errors.New("rule must set at most one of newest or olderThan")
	ErrNegativeOlderThan = errors.New("olderThan must be positive")
	ErrEmptyTagPattern   = errors.New("tagPattern must not be empty")
)
