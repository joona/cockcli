package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Instance maps an alias to connection details.

type Instance struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"apiKey"`
}

type Config struct {
	APIKey    string              `yaml:"apiKey"`    // global default
	Instances map[string]Instance `yaml:"instances"` // keyed by alias
}

// Load reads ~/.config/cockcli/config.yaml or $COCKCLI_CONFIG if set.
func Load() (*Config, error) {
	path := os.Getenv("COCKCLI_CONFIG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".config", "cockcli", "config.yaml")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if len(cfg.Instances) == 0 {
		return nil, errors.New("no instances defined in config file")
	}
	return &cfg, nil
}

// Resolve looks up URL and token for the given instance alias, applying fallbacks.
func (c *Config) Resolve(alias string) (url, token string, err error) {
	inst, ok := c.Instances[alias]
	if !ok {
		return "", "", errors.New("instance alias not found in config: " + alias)
	}
	url = inst.URL
	token = inst.APIKey
	if token == "" {
		token = c.APIKey // fall back to global default
	}
	return
}
