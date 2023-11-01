package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
)

func NewRequestLogger() *slog.Logger {
	b := make([]byte, 4)
	rand.Read(b)
	requestID := hex.EncodeToString(b)
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})).
		With(slog.String("process", "request"), slog.String("id", requestID))
}

func RequestLogger(logger *slog.Logger, r *http.Request) {
	logger.Info(
		"",
		slog.String("path", r.URL.String()),
		slog.String("method", r.Method),
		slog.String("user_agent", r.Header.Get("User-Agent")),
	)
}
