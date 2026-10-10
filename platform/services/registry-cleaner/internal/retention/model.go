package retention

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

type Tag struct {
	Name     string
	PushedAt time.Time
}

type Policy struct {
	Repositories []string `yaml:"repositories" validate:"min=1,dive,required"`
	Rules        []Rule   `yaml:"rules"        validate:"min=1,dive"`
}

type Action string

const (
	ActionKeep   Action = "keep"
	ActionDelete Action = "delete"
)

type Rule struct {
	Priority    int      `yaml:"priority"    validate:"gt=0"`
	Description string   `yaml:"description"`
	Action      Action   `yaml:"action"      validate:"oneof=keep delete"`
	TagPattern  Pattern  `yaml:"tagPattern"`
	Newest      int      `yaml:"newest"      validate:"gte=0"`
	OlderThan   Duration `yaml:"olderThan"`
}

type Reason string

const (
	ReasonRule            Reason = "rule"
	ReasonNoMatchingRule  Reason = "noMatchingRule"
	ReasonUnknownPushDate Reason = "unknownPushDate"
)

type Decision struct {
	Tag    Tag
	Reason Reason
	Rule   *Rule
}

type Plan struct {
	Keep   []Decision
	Delete []Decision
}

const hoursPerDay = 24

const daySuffix = "d"

type Duration struct {
	time.Duration
}

func ParseDuration(value string) (Duration, error) {
	if days, found := strings.CutSuffix(value, daySuffix); found {
		count, err := strconv.Atoi(days)
		if err != nil {
			return Duration{}, fmt.Errorf("invalid duration %q: %w", value, err)
		}
		return Duration{time.Duration(count) * hoursPerDay * time.Hour}, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return Duration{}, fmt.Errorf("invalid duration %q: %w", value, err)
	}
	return Duration{parsed}, nil
}

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	parsed, err := ParseDuration(node.Value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

type Pattern struct {
	*regexp.Regexp
}

func (p *Pattern) UnmarshalYAML(node *yaml.Node) error {
	if node.Value == "" {
		return ErrEmptyTagPattern
	}
	compiled, err := regexp.Compile(node.Value)
	if err != nil {
		return fmt.Errorf("invalid tag pattern %q: %w", node.Value, err)
	}
	p.Regexp = compiled
	return nil
}
