package common

import (
	"log/slog"
	"os"
)

func LogLevelToSlog(level string) slog.Level {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	}
	return slog.LevelInfo
}

func MainLogger() *slog.Logger {
	logLevel := os.Getenv("LOG_LEVEL")
	return slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: LogLevelToSlog(logLevel)}),
	).With(slog.String("process", "main"))
}
