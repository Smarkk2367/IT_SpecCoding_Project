package config

import "os"

type APIConfig struct {
	HTTPAddr    string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	AppEnv      string
}

type WorkerConfig struct {
	HealthAddr     string
	DatabaseURL    string
	RedisURL        string
	ConsumerGroup   string
	ConsumerName    string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	PDFStoragePath  string
	GeoIPProvider   string
	AppEnv          string
}

func LoadAPI() APIConfig {
	return APIConfig{
		HTTPAddr:    env("HTTP_ADDR", ":3000"),
		DatabaseURL: env("DATABASE_URL", "postgres://trackflow:trackflow@localhost:5432/trackflow?sslmode=disable"),
		RedisURL:    env("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:   env("JWT_SECRET", "dev-only-change-me"),
		AppEnv:      env("APP_ENV", "development"),
	}
}

func LoadWorker() WorkerConfig {
	return WorkerConfig{
		HealthAddr:    env("WORKER_HEALTH_ADDR", ":3001"),
		DatabaseURL:   env("DATABASE_URL", "postgres://trackflow:trackflow@localhost:5432/trackflow?sslmode=disable"),
		RedisURL:      env("REDIS_URL", "redis://localhost:6379/0"),
		ConsumerGroup: env("REDIS_CONSUMER_GROUP", "trackflow-workers"),
		ConsumerName:  env("REDIS_CONSUMER_NAME", "worker-1"),
		SMTPHost:      env("SMTP_HOST", "localhost"),
		SMTPPort:      env("SMTP_PORT", "1025"),
		SMTPUser:      os.Getenv("SMTP_USER"),
		SMTPPass:      os.Getenv("SMTP_PASS"),
		SMTPFrom:      env("SMTP_FROM", "noreply@trackflow.io"),
		PDFStoragePath: env("PDF_STORAGE_PATH", "/data/reports"),
		GeoIPProvider: env("GEOIP_PROVIDER", "geoip2"),
		AppEnv:        env("APP_ENV", "development"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
