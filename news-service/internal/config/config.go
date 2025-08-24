package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	OpenAI   OpenAIConfig
	Trending TrendingConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type OpenAIConfig struct {
	APIKey string
	Model  string
}

type TrendingConfig struct {
	TTL           time.Duration
	WorkerInterval time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getEnvAsDuration("READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvAsDuration("IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL: getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/news_system?sslmode=disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		OpenAI: OpenAIConfig{
			APIKey: getEnv("OPENAI_API_KEY", ""),
			Model:  getEnv("LLM_MODEL", "gpt-4o-mini"),
		},
		Trending: TrendingConfig{
			TTL:            getEnvAsDuration("TRENDING_TTL", 120*time.Second),
			WorkerInterval: getEnvAsDuration("TRENDING_WORKER_INTERVAL", 60*time.Second),
		},
	}

	if cfg.OpenAI.APIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

