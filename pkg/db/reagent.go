package db

import (
	"fmt"
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
	Instances int       `json:"instances"`
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
	cols := "reagent.id, reagent.created_at, reagent.updated_at, reagent.name, reagent.formula, COUNT(reagent_instance)"
	join := "LEFT JOIN reagent_instance ON reagent.id = reagent_instance.reagent AND reagent_instance.used IS FALSE"
	filter := "reagent.name ILIKE $3 OR reagent.formula ILIKE $3"
	order := "COUNT(reagent_instance) DESC, reagent.name"
	if len(r.Src) >= 1 {
		query := fmt.Sprintf(
			"SELECT %s FROM reagent %s WHERE %s GROUP BY reagent.id ORDER BY %s LIMIT $1 OFFSET $2",
			cols,
			join,
			filter,
			order,
		)
		batch.Queue(query, r.Limit, r.Offset, r.Src+"%")
	} else {
		query := fmt.Sprintf("SELECT %s FROM reagent %s GROUP BY reagent.id ORDER BY %s LIMIT $1 OFFSET $2", cols, join, order)
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
			&reagent.Instances,
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
	Storage         Storage
	StorageCell     StorageCell
}

type ReagentInstanceRange struct {
	ReagentInstancesExtended []ReagentInstanceExtended
	ReagentID                uuid.UUID
	Limit                    int
	Offset                   int
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
		r.Storage.ID,
		r.StorageCell.Number,
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
	cols := "reagent_instance.id, reagent_instance.created_at, reagent_instance.updated_at, reagent_instance.reagent, reagent_instance.used, reagent_instance.used_at, reagent_instance.expires_at, reagent_instance.storage_cell, storage_cell.id, storage_cell.created_at, storage_cell.updated_at, storage_cell.storage, storage_cell.number, storage.id, storage.created_at, storage.updated_at, storage.name, storage.cells"
	join := "LEFT JOIN storage_cell ON reagent_instance.storage_cell = storage_cell.id LEFT JOIN storage ON storage_cell.storage = storage.id "
	filter := "reagent_instance.reagent=$1"
	order := "reagent_instance.created_at"
	query := fmt.Sprintf(
		"SELECT %s FROM reagent_instance %s WHERE %s ORDER BY %s DESC LIMIT $2 OFFSET $3",
		cols,
		join,
		filter,
		order,
	)
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
		var i ReagentInstanceExtended
		var usedAtVal pgtype.TimestamptzValuer
		err = rows.Scan(
			&i.ReagentInstance.ID,
			&i.ReagentInstance.CreatedAt,
			&i.ReagentInstance.UpdatedAt,
			&i.ReagentInstance.Reagent,
			&i.ReagentInstance.Used,
			usedAtVal,
			&i.ReagentInstance.ExpiresAt,
			&i.ReagentInstance.StorageCell,
			&i.StorageCell.ID,
			&i.StorageCell.CreatedAt,
			&i.StorageCell.UpdatedAt,
			&i.StorageCell.Storage,
			&i.StorageCell.Number,
			&i.Storage.ID,
			&i.Storage.CreatedAt,
			&i.Storage.UpdatedAt,
			&i.Storage.Name,
			&i.Storage.Cells,
		)
		if err != nil {
			return err
		}
		if usedAtVal != nil {
			usedAtPgxTz, err := usedAtVal.TimestamptzValue()
			if err == nil {
				i.ReagentInstance.UsedAt = usedAtPgxTz.Time
			}
		}
		r.ReagentInstancesExtended = append(r.ReagentInstancesExtended, i)
		next = rows.Next()
	}
	return nil
}

func (r *ReagentInstanceRange) Get() (BatchOperation, BatchRead) {
	return r.getQueue, r.getResult
}
