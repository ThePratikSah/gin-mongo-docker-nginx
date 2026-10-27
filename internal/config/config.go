// Package config loads application configuration from environment variables.
// It has no dependencies on other internal packages.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env   string
	HTTP  HTTPConfig
	Mongo MongoConfig
}

type HTTPConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	// TrustedProxies are the CIDRs/IPs of reverse proxies (nginx) whose
	// X-Forwarded-For header Gin is allowed to trust.
	TrustedProxies []string
}

type MongoConfig struct {
	URI            string
	Database       string
	ConnectTimeout time.Duration
}

func Load() (Config, error) {
	var errs []error

	dur := func(key string, def time.Duration) time.Duration {
		v := os.Getenv(key)
		if v == "" {
			return def
		}
		d, err := time.ParseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
			return def
		}
		return d
	}

	cfg := Config{
		Env: env("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Port:            env("HTTP_PORT", "8080"),
			ReadTimeout:     dur("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    dur("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     dur("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: dur("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			TrustedProxies:  csv(env("TRUSTED_PROXIES", "127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")),
		},
		Mongo: MongoConfig{
			URI:            env("MONGO_URI", "mongodb://localhost:27017"),
			Database:       env("MONGO_DATABASE", "ginmongo"),
			ConnectTimeout: dur("MONGO_CONNECT_TIMEOUT", 10*time.Second),
		},
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func csv(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
