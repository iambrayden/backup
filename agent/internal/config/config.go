package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerURL    string `yaml:"server_url"`
	EnrollToken  string `yaml:"enroll_token"`
	DeviceToken  string `yaml:"device_token"`
	DeviceID     string `yaml:"device_id"`
	WatchPath    string `yaml:"watch_path"`
	ConfigPath   string `yaml:"-"`
	AgentVersion string `yaml:"-"`
}

const defaultVersion = "0.1.0"

func Load(path string) (*Config, error) {
	cfg := &Config{
		ConfigPath:   path,
		AgentVersion: defaultVersion,
		ServerURL:    "http://localhost:8080",
		WatchPath:    "/tmp/backup-test",
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.ConfigPath = path
	cfg.AgentVersion = defaultVersion
	return cfg, nil
}

func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.ConfigPath, data, 0600)
}

func (c *Config) IsEnrolled() bool {
	return c.DeviceToken != "" && c.DeviceID != ""
}
