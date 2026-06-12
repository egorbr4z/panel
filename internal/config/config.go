// Package config loads application configuration from environment variables
// (and sensible defaults). Keeping config tiny and env-driven keeps the
// single-binary deployment simple on a 1 GB / 1 vCPU host.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the panel.
type Config struct {
	// HTTP
	HTTPHost string
	HTTPPort int

	// Database: driver is "sqlite" (default) or "postgres".
	DBDriver string
	DBDSN    string // for sqlite this is a file path; for postgres a DSN

	// Auth
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Paths
	DataDir string // base dir for db, generated core configs, certs

	// Core binaries
	XrayBin    string
	SingboxBin string

	// Scheduler
	PollInterval  time.Duration // how often to poll core stats
	FlushInterval time.Duration // how often to flush accumulated usage to DB

	// Subscription
	SubBaseURL string // e.g. https://panel.example.com ; used to build sub links
}

// Load reads configuration from the environment, applying defaults.
func Load() *Config {
	c := &Config{
		HTTPHost:        getEnv("VPANEL_HTTP_HOST", "0.0.0.0"),
		HTTPPort:        getEnvInt("VPANEL_HTTP_PORT", 8080),
		DBDriver:        getEnv("VPANEL_DB", "sqlite"),
		DBDSN:           getEnv("VPANEL_DB_DSN", ""),
		JWTSecret:       getEnv("VPANEL_JWT_SECRET", ""),
		AccessTokenTTL:  getEnvDuration("VPANEL_ACCESS_TTL", 15*time.Minute),
		RefreshTokenTTL: getEnvDuration("VPANEL_REFRESH_TTL", 720*time.Hour),
		DataDir:         getEnv("VPANEL_DATA_DIR", "/var/lib/vpanel"),
		XrayBin:         getEnv("VPANEL_XRAY_BIN", "xray"),
		SingboxBin:      getEnv("VPANEL_SINGBOX_BIN", "sing-box"),
		PollInterval:    getEnvDuration("VPANEL_POLL_INTERVAL", 10*time.Second),
		FlushInterval:   getEnvDuration("VPANEL_FLUSH_INTERVAL", 60*time.Second),
		SubBaseURL:      getEnv("VPANEL_SUB_BASE_URL", ""),
	}

	// Default sqlite path lives under the data dir.
	if c.DBDriver == "sqlite" && c.DBDSN == "" {
		c.DBDSN = c.DataDir + "/vpanel.db"
	}

	// Auto-generate a JWT secret if none supplied. Persisted secrets should be
	// set via env in production so tokens survive restarts.
	if c.JWTSecret == "" {
		c.JWTSecret = randomHex(32)
	}

	return c
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// rand.Read should never fail; fall back to a fixed dev secret.
		return "insecure-dev-secret-change-me"
	}
	return hex.EncodeToString(b)
}
