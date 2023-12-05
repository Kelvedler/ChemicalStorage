package db

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

func NewConnectionPool(
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

type (
	BatchOperation func(batch *pgx.Batch)
	BatchRead      func(results pgx.BatchResults) error
	BatchSet       func() (opeartion BatchOperation, read BatchRead)
)

func PerformBatch(ctx context.Context, dbpool *pgxpool.Pool, batchSets []BatchSet) (errs []error) {
	batch := pgx.Batch{}
	var batchReads []BatchRead
	for _, item := range batchSets {
		operation, read := item()
		operation(&batch)
		batchReads = append(batchReads, read)
	}
	results := dbpool.SendBatch(ctx, &batch)
	for _, read := range batchReads {
		errs = append(errs, read(results))
	}
	results.Close()
	return errs
}

func pgTypeToTime(pgTs pgtype.Timestamptz) (t time.Time) {
	pgxTsVal, err := pgTs.TimestamptzValue()
	if err == nil {
		t = pgxTsVal.Time
	}
	return t
}
