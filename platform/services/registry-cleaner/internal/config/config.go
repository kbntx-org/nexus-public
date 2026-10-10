package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"go.yaml.in/yaml/v3"

	"github.com/kbntx-org/nexus/platform/services/registry-cleaner/internal/retention"
)

const defaultLogLevel = "info"

var ErrInvalidConfig = errors.New("invalid config")

type Config struct {
	LogLevel  string             `yaml:"logLevel"`
	DryRun    bool               `yaml:"dryRun"`
	DockerHub DockerHub          `yaml:"dockerHub"`
	Policies  []retention.Policy `yaml:"policies"  validate:"min=1,dive"`
}

type DockerHub struct {
	Namespace string `yaml:"namespace" validate:"required"`
	Username  string `yaml:"-"         env:"DOCKER_HUB_USERNAME,required,notEmpty"`
	Token     string `yaml:"-"         env:"DOCKER_HUB_TOKEN,required,notEmpty"`
}

type environment struct {
	ConfigPath string `env:"REGISTRY_CLEANER_CONFIG" envDefault:"./config.yaml"`
}

func Load() (*Config, error) {
	settings, err := env.ParseAs[environment]()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(settings.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", settings.ConfigPath, err)
	}
	appConfig, err := Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("config %s: %w", settings.ConfigPath, err)
	}
	if err := env.Parse(&appConfig.DockerHub); err != nil {
		return nil, err
	}
	return appConfig, nil
}

func Parse(raw []byte) (*Config, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	appConfig := Config{LogLevel: defaultLogLevel}
	if err := decoder.Decode(&appConfig); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if err := validate(&appConfig); err != nil {
		return nil, err
	}
	if err := retention.ValidatePolicies(appConfig.Policies); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return &appConfig, nil
}

func validate(appConfig *Config) error {
	structValidator := validator.New(validator.WithRequiredStructEnabled())
	structValidator.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name == "-" {
			return ""
		}
		return name
	})
	err := structValidator.Struct(appConfig)
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err
	}
	failures := make([]string, 0, len(validationErrors))
	for _, fieldError := range validationErrors {
		failures = append(failures, fmt.Sprintf("%s (rule %q)", fieldError.Namespace(), fieldError.Tag()))
	}
	return fmt.Errorf("%w: %s", ErrInvalidConfig, strings.Join(failures, ", "))
}
