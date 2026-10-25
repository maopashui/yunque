package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	DBPath          string
	MigrationsDir   string
	AssetsDir       string
	StaticDir       string
	MaxAssetBytes   int64
	SessionTTL      time.Duration
	CookieSecure    bool
	CookieName      string
	Version         string
	AllowedOrigin   string
	SessionHTTPOnly bool
}

func Load() Config {
	return Config{
		Addr:            env("YUNQUE_ADDR", ":8080"),
		DBPath:          env("YUNQUE_DB_PATH", "data/yunque.db"),
		MigrationsDir:   env("YUNQUE_MIGRATIONS_DIR", "migrations"),
		AssetsDir:       env("YUNQUE_ASSETS_DIR", "data/assets"),
		StaticDir:       env("YUNQUE_STATIC_DIR", ""),
		MaxAssetBytes:   envInt64("YUNQUE_MAX_ASSET_BYTES", 5<<20),
		SessionTTL:      envDuration("YUNQUE_SESSION_TTL", 7*24*time.Hour),
		CookieSecure:    envBool("YUNQUE_COOKIE_SECURE", false),
		CookieName:      env("YUNQUE_COOKIE_NAME", "yunque_session"),
		Version:         env("YUNQUE_VERSION", "0.1.0"),
		// Empty string disables CORS headers (same-origin Docker / production).
		AllowedOrigin:   envOr("YUNQUE_ALLOWED_ORIGIN", "http://localhost:5173"),
		SessionHTTPOnly: true,
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envOr returns the env value even when empty (set but blank), else fallback.
func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
