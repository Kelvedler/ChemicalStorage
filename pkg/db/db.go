package db

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

func GetConnectionPool(
	ctx context.Context,
	mainLogger *slog.Logger,
) *pgxpool.Pool {
	dbpool, err := pgxpool.New(ctx, env.Env.DatabaseUrl)
	if err != nil {
		mainLogger.Error(err.Error())
		os.Exit(1)
	}
	return dbpool
}
