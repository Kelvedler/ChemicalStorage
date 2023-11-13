package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Reagent struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"       validate:"gte=3,lte=300" uaLocal:"назва"`
	Formula   string    `json:"formula"    validate:"gte=1,lte=50"  uaLocal:"формула"`
}

func ReagentCreate(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	newReagent Reagent,
) (Reagent, error) {
	query := "INSERT into reagent(name, formula) VALUES($1, $2) RETURNING id, created_at, updated_at, name, formula"
	var createdReagent Reagent
	err := dbpool.QueryRow(
		ctx,
		query,
		newReagent.Name,
		newReagent.Formula,
	).Scan(
		&createdReagent.ID,
		&createdReagent.CreatedAt,
		&createdReagent.UpdatedAt,
		&createdReagent.Name,
		&createdReagent.Formula,
	)
	return createdReagent, err
}

func reagentGetSlice(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	query string,
	args ...any,
) ([]Reagent, error) {
	reagentsSlice := make([]Reagent, 0)
	rows, err := dbpool.Query(ctx, query, args...)
	if err != nil {
		return reagentsSlice, err
	}
	next := rows.Next()
	if !next {
		return reagentsSlice, nil
	}
	for next {
		var reagent Reagent
		rows.Scan(
			&reagent.ID,
			&reagent.CreatedAt,
			&reagent.UpdatedAt,
			&reagent.Name,
			&reagent.Formula,
		)
		reagentsSlice = append(reagentsSlice, reagent)
		next = rows.Next()
	}
	return reagentsSlice, nil
}

func ReagentGetRange(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	limit int,
	offset int,
	src string,
) ([]Reagent, error) {
	orderBy := "-created_at"
	if len(src) >= 1 {
		query := "SELECT * FROM reagent WHERE name ILIKE $4 OR formula ILIKE $4 ORDER BY $1 LIMIT $2 OFFSET $3"
		return reagentGetSlice(ctx, dbpool, query, orderBy, limit, offset, src+"%")
	} else {
		query := "SELECT * FROM reagent ORDER BY $1 LIMIT $2 OFFSET $3"
		return reagentGetSlice(ctx, dbpool, query, orderBy, limit, offset)
	}
}

func ReagentGet(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	id string,
) (reagent Reagent, err error) {
	query := "SELECT * FROM reagent WHERE id=$1"
	err = dbpool.QueryRow(ctx, query, id).
		Scan(&reagent.ID, &reagent.CreatedAt, &reagent.UpdatedAt, &reagent.Name, &reagent.Formula)
	return reagent, err
}

type ReagentInstanceFull struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Reagent   uuid.UUID `json:"reagent"`
	Used      bool      `json:"used"`
	UsedAt    bool      `json:"used_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
