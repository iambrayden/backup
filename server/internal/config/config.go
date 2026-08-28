package config

import (
	"encoding/base64"
	"errors"
	"os"
)

type Config struct {
	DatabaseURL    string
	RedisURL       string
	MasterKey      []byte
	JWTSecret      []byte
	PlatformDomain string
	AdminEmail     string
	AdminPassword  string
	HTTPAddr       string
}

func Load() (*Config, error) {
	c := &Config{
		DatabaseURL:    env("DATABASE_URL", "postgres://backup:backup@localhost:5432/backup?sslmode=disable"),
		RedisURL:       env("REDIS_URL", "redis://localhost:6379/0"),
		PlatformDomain: env("PLATFORM_DOMAIN", "localhost"),
		AdminEmail:     env("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:  env("ADMIN_PASSWORD", "changeme"),
		HTTPAddr:       env("HTTP_ADDR", ":8080"),
	}

	mk, err := decodeKey("MASTER_KEY")
	if err != nil {
		return nil, err
	}
	c.MasterKey = mk

	jk, err := decodeKey("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	c.JWTSecret = jk

	return c, nil
}

func decodeKey(envVar string) ([]byte, error) {
	raw := os.Getenv(envVar)
	if raw == "" {
		return nil, errors.New(envVar + " is required")
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, errors.New(envVar + ": " + err.Error())
	}
	if len(b) != 32 {
		return nil, errors.New(envVar + " must decode to exactly 32 bytes")
	}
	return b, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
