package db

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type StorageInput struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Name      string `json:"name"`
	Cells     string `json:"cells"`
}

type Storage struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"       validate:"gte=3,lte=100"  uaLocal:"назва"`
	Cells     int16     `json:"cells"      validate:"gte=1,lte=1000" uaLocal:"відділи"`
}

type StoragesRange struct {
	Storages []Storage
	Limit    int
	Offset   int
	Src      string
}

func (input StorageInput) Bind() (output Storage, err error) {
	if input.ID != "" {
		id, err := uuid.Parse(input.ID)
		if err != nil {
			return Storage{}, err
		}
		output.ID = id
	}
	if input.CreatedAt != "" {
		createdAt, err := strconv.Atoi(input.CreatedAt)
		if err != nil {
			return Storage{}, err
		}
		output.CreatedAt = time.UnixMilli(int64(createdAt)).UTC()
	}
	if input.UpdatedAt != "" {
		updatedAt, err := strconv.Atoi(input.UpdatedAt)
		if err != nil {
			return Storage{}, err
		}
		output.UpdatedAt = time.UnixMilli(int64(updatedAt)).UTC()
	}
	output.Name = input.Name
	if input.Cells != "" {
		cells, err := strconv.Atoi(input.Cells)
		if err != nil {
			return Storage{}, err
		}
		output.Cells = int16(cells)
	}
	return output, nil
}

func (s Storage) createQueue(
	batch *pgx.Batch,
) {
	query := "INSERT INTO storage(name, cells) VALUES($1, $2) RETURNING id, created_at, updated_at"
	batch.Queue(query, s.Name, s.Cells)
}

func (s *Storage) createResult(results pgx.BatchResults) error {
	return results.QueryRow().Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (s *Storage) Create() (BatchOperation, BatchRead) {
	return s.createQueue, s.createResult
}

func (s StoragesRange) getQueue(
	batch *pgx.Batch,
) {
	if len(s.Src) >= 1 {
		query := "SELECT * FROM storage WHERE name ILIKE=$3 ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		batch.Queue(query, s.Limit, s.Offset, s.Src+"%")
	} else {
		query := "SELECT * FROM storage ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		batch.Queue(query, s.Limit, s.Offset)
	}
}

func (s *StoragesRange) getResult(results pgx.BatchResults) error {
	rows, err := results.Query()
	if err != nil {
		return err
	}
	next := rows.Next()
	if !next {
		return nil
	}
	for next {
		var storage Storage
		err = rows.Scan(
			&storage.ID,
			&storage.CreatedAt,
			&storage.UpdatedAt,
			&storage.Name,
			&storage.Cells,
		)
		if err != nil {
			return err
		}
		s.Storages = append(s.Storages, storage)
		next = rows.Next()
	}
	return nil
}

func (s *StoragesRange) Get() (BatchOperation, BatchRead) {
	return s.getQueue, s.getResult
}

func (s Storage) getQueue(
	batch *pgx.Batch,
) {
	query := "SELECT created_at, updated_at, name FROM storage WHERE id=$1"
	batch.Queue(query, s.ID)
}

func (s *Storage) getResult(results pgx.BatchResults) error {
	return results.QueryRow().Scan(&s.CreatedAt, &s.UpdatedAt, &s.Name)
}

func (s *Storage) Get() (BatchOperation, BatchRead) {
	return s.getQueue, s.getResult
}

type StorageCell struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Storage   uuid.UUID `json:"storage"`
	Number    int16     `json:"number"     uaLocal:"номер"`
}

func (storageCell StorageCell) tryCreateQueue(
	batch *pgx.Batch,
) {
	query := "INSERT INTO storage_cell(storage, number) VALUES($1, $2) ON CONFLICT ON CONSTRAINT storage_cell_storage_number_key DO NOTHING"
	batch.Queue(query, storageCell.Storage, storageCell.Number)
}

func (storageCell *StorageCell) tryCreateResult(results pgx.BatchResults) error {
	_, err := results.Exec()
	return err
}

func (storageCell *StorageCell) TryCreate() (BatchOperation, BatchRead) {
	return storageCell.tryCreateQueue, storageCell.tryCreateResult
}
