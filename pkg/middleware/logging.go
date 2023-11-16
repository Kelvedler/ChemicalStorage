package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		a.Value = slog.TimeValue(time.Now().UTC())
	}
	return a
}

func NewRequestLogger() *slog.Logger {
	b := make([]byte, 4)
	rand.Read(b)
	requestID := hex.EncodeToString(b)
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: replaceAttr, Level: slog.LevelDebug})).
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
