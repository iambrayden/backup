package config

import (
	"encoding/base64"
	"errors"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
	RedisURL       string
	MasterKey      []byte
	JWTSecret      []byte
	PlatformDomain string
	CORSOrigins    []string
	AdminEmail     string
	AdminPassword  string
	HTTPAddr       string
}

func Load() (*Config, error) {
	domain := env("PLATFORM_DOMAIN", "localhost")

	c := &Config{
		DatabaseURL:    env("DATABASE_URL", "postgres://backup:backup@localhost:5432/backup?sslmode=disable"),
		RedisURL:       env("REDIS_URL", "redis://localhost:6379/0"),
		PlatformDomain: domain,
		AdminEmail:     env("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:  env("ADMIN_PASSWORD", "changeme"),
		HTTPAddr:       env("HTTP_ADDR", ":8080"),
	}

	if raw := os.Getenv("CORS_ORIGINS"); raw != "" {
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				c.CORSOrigins = append(c.CORSOrigins, o)
			}
		}
	} else {
		c.CORSOrigins = []string{
			"https://" + domain,
			"http://localhost",
			"http://localhost:3000",
		}
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
