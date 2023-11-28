package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Reagent struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"       validate:"gte=3,lte=300" uaLocal:"назва"`
	Formula   string    `json:"formula"    validate:"gte=1,lte=50"  uaLocal:"формула"`
}

type ReagentsRange struct {
	Reagents []Reagent
	Limit    int
	Offset   int
	Src      string
}

func (r Reagent) createQueue(
	batch *pgx.Batch,
) {
	query := "INSERT into reagent(name, formula) VALUES($1, $2) RETURNING id, created_at, updated_at"
	batch.Queue(query, r.Name, r.Formula)
}

func (r *Reagent) createResult(results pgx.BatchResults) error {
	return results.QueryRow().Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
}

func (r *Reagent) Create() (BatchOperation, BatchRead) {
	return r.createQueue, r.createResult
}

func (r ReagentsRange) getQueue(
	batch *pgx.Batch,
) {
	if len(r.Src) >= 1 {
		query := "SELECT * FROM reagent WHERE name ILIKE $3 OR formula ILIKE $3 ORDER BY name LIMIT $1 OFFSET $2"
		batch.Queue(query, r.Limit, r.Offset, r.Src+"%")
	} else {
		query := "SELECT * FROM reagent ORDER BY name LIMIT $1 OFFSET $2"
		batch.Queue(query, r.Limit, r.Offset)
	}
}

func (r *ReagentsRange) getResult(results pgx.BatchResults) error {
	rows, err := results.Query()
	if err != nil {
		return err
	}
	next := rows.Next()
	if !next {
		return nil
	}
	for next {
		var reagent Reagent
		err = rows.Scan(
			&reagent.ID,
			&reagent.CreatedAt,
			&reagent.UpdatedAt,
			&reagent.Name,
			&reagent.Formula,
		)
		if err != nil {
			return err
		}
		r.Reagents = append(r.Reagents, reagent)
		next = rows.Next()
	}
	return nil
}

func (r *ReagentsRange) Get() (BatchOperation, BatchRead) {
	return r.getQueue, r.getResult
}

func (reagent Reagent) getQueue(
	batch *pgx.Batch,
) {
	query := "SELECT created_at, updated_at, name, formula FROM reagent WHERE id=$1"
	batch.Queue(query, reagent.ID)
}

func (reagent *Reagent) getResult(results pgx.BatchResults) error {
	return results.QueryRow().Scan(
		&reagent.CreatedAt,
		&reagent.UpdatedAt,
		&reagent.Name,
		&reagent.Formula,
	)
}

func (reagent *Reagent) Get() (BatchOperation, BatchRead) {
	return reagent.getQueue, reagent.getResult
}

func (r Reagent) updateQueue(
	batch *pgx.Batch,
) {
	query := "UPDATE reagent SET name=$2, formula=$3 WHERE id=$1"
	batch.Queue(query, r.ID, r.Name, r.Formula)
}

func (r *Reagent) updateResult(results pgx.BatchResults) error {
	result, err := results.Exec()
	affectedRows := result.RowsAffected()
	if err != nil {
		return err
	} else if affectedRows != 1 {
		if affectedRows == 0 {
			return pgx.ErrNoRows
		}
	}
	return nil
}

func (r *Reagent) Update() (BatchOperation, BatchRead) {
	return r.updateQueue, r.updateResult
}

type ReagentInstance struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Reagent     uuid.UUID `json:"reagent"`
	Used        bool      `json:"used"`
	UsedAt      time.Time `json:"used_at"`
	ExpiresAt   time.Time `json:"expires_at"   validate:"gt" uaLocal:"термін придатності"`
	StorageCell uuid.UUID `json:"storage_cell"`
}

type ReagentInstanceExtended struct {
	ReagentInstance ReagentInstance
	Storage         uuid.UUID
	CellNumber      int16
}

type ReagentInstanceRange struct {
	ReagentInstances []ReagentInstance
	ReagentID        uuid.UUID
	Limit            int
	Offset           int
}

func (r *ReagentInstanceExtended) createQueue(
	batch *pgx.Batch,
) {
	reagentInstance := r.ReagentInstance
	query := "INSERT INTO reagent_instance(reagent, expires_at, storage_cell) VALUES($1, $2, (SELECT id FROM storage_cell WHERE storage=$3 AND number=$4)) RETURNING id, created_at, updated_at"
	batch.Queue(
		query,
		reagentInstance.Reagent,
		reagentInstance.ExpiresAt,
		r.Storage,
		r.CellNumber,
	)
}

func (r *ReagentInstance) createResult(results pgx.BatchResults) error {
	return results.QueryRow().Scan(
		&r.ID,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
}

func (r *ReagentInstanceExtended) Create() (BatchOperation, BatchRead) {
	return r.createQueue, r.ReagentInstance.createResult
}

func (r *ReagentInstanceRange) getQueue(
	batch *pgx.Batch,
) {
	query := "SELECT * FROM reagent_instance WHERE reagent=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	batch.Queue(query, r.ReagentID, r.Limit, r.Offset)
}

func (r *ReagentInstanceRange) getResult(results pgx.BatchResults) error {
	rows, err := results.Query()
	if err != nil {
		return err
	}
	next := rows.Next()
	if !next {
		return nil
	}
	for next {
		var reagentInstance ReagentInstance
		var usedAtVal pgtype.TimestamptzValuer
		err = rows.Scan(
			&reagentInstance.ID,
			&reagentInstance.CreatedAt,
			&reagentInstance.UpdatedAt,
			&reagentInstance.Reagent,
			&reagentInstance.Used,
			usedAtVal,
			&reagentInstance.ExpiresAt,
			&reagentInstance.StorageCell,
		)
		if err != nil {
			return err
		}
		if usedAtVal != nil {
			usedAtPgxTz, err := usedAtVal.TimestamptzValue()
			if err == nil {
				reagentInstance.UsedAt = usedAtPgxTz.Time
			}
		}
		r.ReagentInstances = append(r.ReagentInstances, reagentInstance)
		next = rows.Next()
	}
	return nil
}

func (r *ReagentInstanceRange) Get() (BatchOperation, BatchRead) {
	return r.getQueue, r.getResult
}
