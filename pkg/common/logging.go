package common

import (
	"log/slog"
	"os"

	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

func MainLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: env.Env.LogLevel}),
	).With(slog.String("process", "main"))
}
