package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kbntx-org/nexus/platform/services/registry-cleaner/internal/retention"
)

const validConfig = `
dryRun: true
dockerHub:
  namespace: kbntx
policies:
  - repositories: [smelt, kiln]
    rules:
      - priority: 1
        description: Keep latest
        action: keep
        tagPattern: '^latest$'
      - priority: 2
        action: keep
        newest: 10
      - priority: 3
        action: delete
        olderThan: 5d
`

const namespace = "dockerHub: {namespace: kbntx}\n"

func TestParse(t *testing.T) {
	appConfig, err := Parse([]byte(validConfig))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !appConfig.DryRun || appConfig.LogLevel != defaultLogLevel {
		t.Errorf("unexpected settings: %+v", appConfig)
	}
	rules := appConfig.Policies[0].Rules
	if rules[0].Action != retention.ActionKeep || rules[0].Description != "Keep latest" {
		t.Errorf("unexpected first rule %+v", rules[0])
	}
	if !rules[0].TagPattern.MatchString("latest") || rules[0].TagPattern.MatchString("latest-pr") {
		t.Errorf("tagPattern %v does not match as expected", rules[0].TagPattern)
	}
	if rules[1].Newest != 10 {
		t.Errorf("newest = %d, expected 10", rules[1].Newest)
	}
	if rules[2].Action != retention.ActionDelete || rules[2].OlderThan.Duration != 5*24*time.Hour {
		t.Errorf("unexpected third rule %+v", rules[2])
	}
}

func TestParseRejectsInvalidConfig(t *testing.T) {
	testCases := []struct {
		name     string
		raw      string
		expected error
	}{
		{
			name:     "missing namespace",
			raw:      "policies: [{repositories: [smelt], rules: [{priority: 1, action: keep}]}]",
			expected: ErrInvalidConfig,
		},
		{
			name:     "no policies",
			raw:      namespace,
			expected: ErrInvalidConfig,
		},
		{
			name:     "no repositories",
			raw:      namespace + "policies: [{rules: [{priority: 1, action: keep}]}]",
			expected: ErrInvalidConfig,
		},
		{
			name:     "no rules",
			raw:      namespace + "policies: [{repositories: [smelt]}]",
			expected: ErrInvalidConfig,
		},
		{
			name:     "missing priority",
			raw:      namespace + "policies: [{repositories: [smelt], rules: [{action: keep}]}]",
			expected: ErrInvalidConfig,
		},
		{
			name:     "missing action",
			raw:      namespace + "policies: [{repositories: [smelt], rules: [{priority: 1}]}]",
			expected: ErrInvalidConfig,
		},
		{
			name:     "unknown action",
			raw:      namespace + "policies: [{repositories: [smelt], rules: [{priority: 1, action: expire}]}]",
			expected: ErrInvalidConfig,
		},
		{
			name:     "negative newest",
			raw:      namespace + "policies: [{repositories: [smelt], rules: [{priority: 1, action: keep, newest: -1}]}]",
			expected: ErrInvalidConfig,
		},
		{
			name:     "duplicate priority",
			raw:      namespace + "policies: [{repositories: [smelt], rules: [{priority: 1, action: keep}, {priority: 1, action: delete}]}]",
			expected: retention.ErrDuplicatePriority,
		},
		{
			name:     "empty tag pattern",
			raw:      namespace + "policies: [{repositories: [smelt], rules: [{priority: 1, action: keep, tagPattern: ''}]}]",
			expected: retention.ErrEmptyTagPattern,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := Parse([]byte(testCase.raw)); !errors.Is(err, testCase.expected) {
				t.Errorf("got %v, expected %v", err, testCase.expected)
			}
		})
	}
}

func TestParseRejectsUnknownKeys(t *testing.T) {
	raw := namespace + "policies: [{repositories: [smelt], rules: [{priority: 1, action: delete, olderthan: 5d}]}]"
	if _, err := Parse([]byte(raw)); err == nil {
		t.Error("expected a typo in a rule to be rejected")
	}
}

func writeConfig(t *testing.T) string {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(validConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadReadsCredentialsFromTheEnvironment(t *testing.T) {
	t.Setenv("REGISTRY_CLEANER_CONFIG", writeConfig(t))
	t.Setenv("DOCKER_HUB_USERNAME", "robot")
	t.Setenv("DOCKER_HUB_TOKEN", "secret")

	appConfig, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if appConfig.DockerHub.Username != "robot" || appConfig.DockerHub.Token != "secret" {
		t.Errorf("unexpected credentials: %+v", appConfig.DockerHub)
	}
}

func TestLoadRequiresCredentials(t *testing.T) {
	t.Setenv("REGISTRY_CLEANER_CONFIG", writeConfig(t))
	t.Setenv("DOCKER_HUB_USERNAME", "robot")
	t.Setenv("DOCKER_HUB_TOKEN", "")

	if _, err := Load(); err == nil {
		t.Error("expected an empty token to be rejected")
	}
}
