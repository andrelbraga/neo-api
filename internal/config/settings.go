package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const (
	EnvLocal      = "local"
	EnvHml        = "hml"
	EnvProduction = "production"

	// devJWTSecret só existe para desenvolvimento local. Fora dele a API não sobe com ele.
	devJWTSecret = "neo360-dev-secret-nao-usar-em-producao"
)

var (
	ErrMissingJWTSecret   = errors.New("JWT_SECRET é obrigatório fora do ambiente local")
	ErrMissingDatabaseURL = errors.New("DATABASE_URL é obrigatório fora do ambiente local")
)

type Configs struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	// DBSchema isola as tabelas num schema próprio quando o banco é dividido com outro projeto (ADR-0018).
	DBSchema       string
	JWTSecret      string
	JWTExpiryHours int
	AllowedOrigins []string
}

// Load lê as variáveis de ambiente. Os defaults servem ao ambiente local (docker compose).
func Load() Configs {
	return Configs{
		AppEnv:         getEnv("APP_ENV", EnvLocal),
		Port:           getEnv("PORT", "8081"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://neo360:neo360@localhost:5433/neo360?sslmode=disable"),
		DBSchema:       getEnv("DB_SCHEMA", ""),
		JWTSecret:      getEnv("JWT_SECRET", devJWTSecret),
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 12),
		AllowedOrigins: splitList(getEnv("ALLOWED_ORIGINS", "http://localhost:5174")),
	}
}

// IsDeployed vale para todo ambiente publicado: só "local" usa os defaults de desenvolvimento.
func (c Configs) IsDeployed() bool {
	return c.AppEnv != EnvLocal
}

// Validate impede que a API suba fora do ambiente local com defaults de desenvolvimento.
func (c Configs) Validate() error {
	if !c.IsDeployed() {
		return nil
	}
	if c.JWTSecret == "" || c.JWTSecret == devJWTSecret {
		return ErrMissingJWTSecret
	}
	if os.Getenv("DATABASE_URL") == "" {
		return ErrMissingDatabaseURL
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
