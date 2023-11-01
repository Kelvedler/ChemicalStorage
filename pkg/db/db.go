package db

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetConnectionPool(ctx context.Context, mainLogger *slog.Logger) *pgxpool.Pool {
	dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		mainLogger.Error(err.Error())
		os.Exit(1)
	}
	return dbpool
}
