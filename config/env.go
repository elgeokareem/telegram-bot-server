package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Env struct {
	AppEnv             string
	Port               string
	GinMode            string
	TelegramBotToken   string
	CORSAllowedOrigins []string
	DBSchema           string
	DBName             string
	DBUser             string
	DBPassword         string
	DBHost             string
	DBPort             string
}

var Current Env

func Init() error {
	env, err := Load()
	if err != nil {
		return err
	}

	Current = env
	return nil
}

func Load() (Env, error) {
	_ = godotenv.Load(".env")

	env := Env{
		AppEnv:             getEnvOrDefault("APP_ENV", "development"),
		Port:               getEnvOrDefault("PORT", "8080"),
		GinMode:            getEnvOrDefault("GIN_MODE", "debug"),
		TelegramBotToken:   strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		CORSAllowedOrigins: splitCSV(getEnvOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:4173,http://127.0.0.1:4173,https://telegram.william-vegas.com")),
		DBSchema:           getEnvOrDefault("DB_SCHEMA", "postgres"),
		DBName:             getEnvOrDefault("DB_NAME", ""),
		DBUser:             getEnvOrDefault("DB_USER", "postgres"),
		DBPassword:         getEnvOrDefault("DB_PASSWORD", ""),
		DBHost:             getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:             getEnvOrDefault("DB_PORT", "5432"),
	}

	if strings.TrimSpace(env.Port) == "" {
		return env, fmt.Errorf("missing required environment variable: PORT")
	}

	if strings.TrimSpace(env.DBName) == "" {
		return env, fmt.Errorf("missing required environment variable: DB_NAME")
	}

	if strings.TrimSpace(env.TelegramBotToken) == "" {
		return env, fmt.Errorf("missing required environment variable: TELEGRAM_BOT_TOKEN")
	}

	return env, nil
}

func getEnvOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func splitCSV(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))

	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
