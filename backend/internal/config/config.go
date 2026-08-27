// package config provides environment-driven application configuration.
// all values are loaded from environment variables with sensible defaults.
// no external config libraries — just stdlib.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Log      LogConfig
	RabbitMQ RabbitMQConfig
	Redis    RedisConfig
}

// serverconfig holds http server settings.
type ServerConfig struct {
	Port            int
	ShutdownTimeout time.Duration
}

// databaseconfig holds postgresql connection settings.
type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

// jwtconfig holds jwt authentication settings.
type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

// logconfig holds logging settings.
type LogConfig struct {
	Level string // debug, info, warn, error
}

// rabbitmqconfig holds message broker settings.
type RabbitMQConfig struct {
	URL string
}

// redisconfig holds redis settings.
type RedisConfig struct {
	URL             string
	RateLimitRPS    int // requests per window
	RateLimitWindow int // window in seconds
}

// load reads configuration from environment variables.
// returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnvInt("SERVER_PORT", 8080),
			ShutdownTimeout: time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 25)),
			MinConns:        int32(getEnvInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: time.Duration(getEnvInt("DB_MAX_CONN_LIFETIME_MINUTES", 30)) * time.Minute,
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
			Expiry: time.Duration(getEnvInt("JWT_EXPIRY_SECONDS", 3600)) * time.Second,
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", ""),
		},
		Redis: RedisConfig{
			URL:             getEnv("REDIS_URL", ""),
			RateLimitRPS:    getEnvInt("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow: getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.RabbitMQ.URL == "" {
		return fmt.Errorf("RABBITMQ_URL is required")
	}
	if c.Redis.URL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
