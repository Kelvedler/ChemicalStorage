package common

import (
	"log/slog"

	"github.com/joho/godotenv"
)

func LoadDotenv(mainLogger *slog.Logger) {
	err := godotenv.Load(".env")
	if err != nil {
		mainLogger.Warn("Failed to load .env file, using os environment instead")
	}
}
