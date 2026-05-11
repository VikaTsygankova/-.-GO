package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort  string
	DatabaseURL string
	JWTSecret   string
	HMACSecret  string
	PGPKey      string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
}

func Load() Config {
	godotenv.Load()
	return Config{
		ServerPort:  getenv("SERVER_PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/bank_service?sslmode=disable"),
		JWTSecret:   getenv("JWT_SECRET", "change-me-super-secret-key-change-me"),
		HMACSecret:  getenv("HMAC_SECRET", "change-me-hmac-secret"),
		PGPKey:      getenv("PGP_KEY", "demo-pgp-key"),
		SMTPHost:    getenv("SMTP_HOST", "smtp.example.com"),
		SMTPPort:    getenvInt("SMTP_PORT", 587),
		SMTPUser:    getenv("SMTP_USER", "noreply@example.com"),
		SMTPPass:    getenv("SMTP_PASS", "password"),
		SMTPFrom:    getenv("SMTP_FROM", "noreply@example.com"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func getenvInt(key string, fallback int) int {
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
