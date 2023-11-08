package env

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func loadDotenv(logger *slog.Logger) {
	err := godotenv.Load(".env")
	if err != nil {
		logger.Warn("Failed to load .env file, using os environment instead")
	}
}

type Config struct {
	SecretKey    string
	LogLevel     slog.Level
	DatabaseUrl  string
	AllowedHosts string
	Port         int
	Jwt          Jwt
}

type Jwt struct {
	Domain                 string
	SecureCookies          bool
	ExpirationDeltaMinutes int
}

var Env Config

func setSecretKey(logger *slog.Logger) {
	envKey := "SECRET_KEY"
	secretKey := os.Getenv(envKey)
	if secretKey == "" {
		logger.Error(fmt.Sprintf("Could not get '%s'", envKey))
		os.Exit(1)
	}
	Env.SecretKey = secretKey
}

func setLogLevel(logger *slog.Logger) {
	envKey := "LOG_LEVEL"
	level := os.Getenv(envKey)
	switch level {
	case "DEBUG":
		Env.LogLevel = slog.LevelDebug
	case "WARN":
		Env.LogLevel = slog.LevelWarn
	case "ERROR":
		Env.LogLevel = slog.LevelError
	default:
		if level != "INFO" {
			logger.Info(fmt.Sprintf("Could not get '%s', set to INFO", envKey))
		}
		Env.LogLevel = slog.LevelInfo
	}
}

func setDatabaseUrl(logger *slog.Logger) {
	envKey := "DATABASE_URL"
	databaseUrl := os.Getenv(envKey)
	if databaseUrl == "" {
		logger.Error(fmt.Sprintf("Could not get '%s'", envKey))
		os.Exit(1)
	}
	Env.DatabaseUrl = databaseUrl
}

func setAllowedHosts(logger *slog.Logger) {
	envKey := "ALLOWED_HOSTS"
	allowedHosts := os.Getenv(envKey)
	if allowedHosts == "" {
		logger.Error(fmt.Sprintf("Could not get '%s'", envKey))
		os.Exit(1)
	}
	Env.AllowedHosts = allowedHosts
}

func setPort(logger *slog.Logger) {
	envKey := "PORT"
	portInt, err := strconv.Atoi(os.Getenv(envKey))
	if err != nil {
		logger.Error(fmt.Sprintf("Could not get '%s'", envKey))
		os.Exit(1)
	}
	Env.Port = portInt
}

func setJwt(logger *slog.Logger) {
	secure, err := strconv.ParseBool(os.Getenv("JWT_SECURE_COOKIES"))
	if err != nil {
		logger.Error("Could not get 'JWT_SECURE_COOKIES'")
		os.Exit(1)
	}
	domain := os.Getenv("JWT_DOMAIN")
	if domain == "" {
		logger.Error("Could not get 'JWT_DOMAIN'")
		os.Exit(1)
	}
	exp, err := strconv.Atoi(os.Getenv("JWT_EXP_DELTA_MINUTES"))
	if err != nil {
		logger.Error("Could not get 'JWT_EXP_DELTA_MINUTES'")
		os.Exit(1)
	}
	jwt := Jwt{
		Domain:                 domain,
		SecureCookies:          secure,
		ExpirationDeltaMinutes: exp,
	}
	Env.Jwt = jwt
}

func InitEnv() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}),
	).With(slog.String("process", "main"))
	loadDotenv(logger)

	setSecretKey(logger)
	setLogLevel(logger)
	setDatabaseUrl(logger)
	setAllowedHosts(logger)
	setPort(logger)
	setJwt(logger)
}
