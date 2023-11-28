package db

import (
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Role struct {
	Name      string
	NameLocal string
}

var (
	Admin = Role{
		Name:      "admin",
		NameLocal: "адміністратор",
	}
	Assistant = Role{
		Name:      "assistant",
		NameLocal: "лаборант",
	}
	Lecturer = Role{
		Name:      "lecturer",
		NameLocal: "викладач",
	}
	Unconfirmed = Role{
		Name:      "unconfirmed",
		NameLocal: "не підтверджений",
	}
)

func StringToRole(roleStr string) (Role, error) {
	roles := []Role{Admin, Assistant, Lecturer, Unconfirmed}
	for _, role := range roles {
		if roleStr == role.Name {
			return role, nil
		}
	}
	return Role{}, RoleInvalid
}

var RoleInvalid = errors.New("Storage user role is not valid")

type StorageUserInput struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Password  string `json:"password"`
	Active    string `json:"active"`
}

type StorageUser struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"       validate:"gte=3,lte=50" uaLocal:"логін"`
	Role      Role      `json:"role"                               uaLocal:"роль"`
	Password  string    `json:"password"   validate:"gte=6,lte=20" uaLocal:"пароль"`
	Active    bool      `json:"active"`
}

type StorageUsersRange struct {
	StorageUsers []StorageUser
	Limit        int
	Offset       int
	Src          string
	ExcludeID    uuid.UUID
}

func (input StorageUserInput) Bind() (output StorageUser, err error) {
	if input.ID != "" {
		id, err := uuid.Parse(input.ID)
		if err != nil {
			return StorageUser{}, err
		}
		output.ID = id
	}
	if input.CreatedAt != "" {
		createdAt, err := strconv.Atoi(input.CreatedAt)
		if err != nil {
			return StorageUser{}, err
		}
		output.CreatedAt = time.UnixMilli(int64(createdAt)).UTC()
	}
	if input.UpdatedAt != "" {
		updatedAt, err := strconv.Atoi(input.UpdatedAt)
		if err != nil {
			return StorageUser{}, err
		}
		output.UpdatedAt = time.UnixMilli(int64(updatedAt)).UTC()
	}
	output.Name = input.Name
	if input.Role != "" {
		role, err := StringToRole(input.Role)
		if err != nil {
			return StorageUser{}, err
		} else {
			output.Role = role
		}
	}
	output.Password = input.Password
	if input.Active == "" {
		input.Active = "false"
	}
	if input.Active != "" {
		active, err := strconv.ParseBool(input.Active)
		if err != nil {
			return StorageUser{}, err
		}
		output.Active = active
	}
	return output, nil
}

func (s StorageUser) createQueue(
	batch *pgx.Batch,
) {
	query := "INSERT into storage_user(name, password, role) VALUES($1, $2, $3) RETURNING id, created_at, updated_at, active"
	batch.Queue(query, s.Name, s.Password, s.Role.Name)
}

func (s *StorageUser) createResult(result pgx.BatchResults) error {
	return result.QueryRow().Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt, &s.Active)
}

func (s *StorageUser) Create() (BatchOperation, BatchRead) {
	return s.createQueue, s.createResult
}

func (s StorageUsersRange) getQueue(
	batch *pgx.Batch,
) {
	if len(s.Src) >= 1 {
		query := "SELECT * FROM storage_user WHERE name ILIKE $3 AND id!=$4 ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		batch.Queue(query, s.Limit, s.Offset, s.Src+"%", s.ExcludeID)
	} else {
		query := "SELECT * FROM storage_user WHERE id!=$3 ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		batch.Queue(query, s.Limit, s.Offset, s.ExcludeID)
	}
}

func (s *StorageUsersRange) getResult(result pgx.BatchResults) error {
	rows, err := result.Query()
	if err != nil {
		return err
	}
	next := rows.Next()
	if !next {
		return nil
	}
	for next {
		var roleStr string
		var storageUser StorageUser
		err = rows.Scan(
			&storageUser.ID,
			&storageUser.CreatedAt,
			&storageUser.UpdatedAt,
			&storageUser.Name,
			&roleStr,
			&storageUser.Password,
			&storageUser.Active,
		)
		if err != nil {
			return err
		}
		role, err := StringToRole(roleStr)
		if err != nil {
			return err
		} else {
			storageUser.Role = role
		}
		s.StorageUsers = append(s.StorageUsers, storageUser)
		next = rows.Next()
	}
	return nil
}

func (s *StorageUsersRange) Get() (BatchOperation, BatchRead) {
	return s.getQueue, s.getResult
}

func (s StorageUser) getByIDQueue(
	batch *pgx.Batch,
) {
	query := "SELECT created_at, updated_at, name, role, password, active FROM storage_user WHERE id=$1"
	batch.Queue(query, s.ID)
}

func (s *StorageUser) getByIDResult(results pgx.BatchResults) (err error) {
	var roleStr string
	err = results.QueryRow().Scan(
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.Name,
		&roleStr,
		&s.Password,
		&s.Active,
	)
	if err != nil {
		return err
	}
	role, err := StringToRole(roleStr)
	if err != nil {
		return err
	}
	s.Role = role
	return nil
}

func (s *StorageUser) GetByID() (BatchOperation, BatchRead) {
	return s.getByIDQueue, s.getByIDResult
}

func (s StorageUser) getByNameQueue(
	batch *pgx.Batch,
) {
	query := "SELECT id, created_at, updated_at, role, password, active FROM storage_user WHERE name=$1"
	batch.Queue(query, s.Name)
}

func (s *StorageUser) getByNameResult(results pgx.BatchResults) (err error) {
	var roleStr string
	err = results.QueryRow().Scan(
		&s.ID,
		&s.CreatedAt,
		&s.UpdatedAt,
		&roleStr,
		&s.Password,
		&s.Active,
	)
	if err != nil {
		return err
	}
	role, err := StringToRole(roleStr)
	if err != nil {
		return err
	}
	s.Role = role
	return nil
}

func (s *StorageUser) GetByName() (BatchOperation, BatchRead) {
	return s.getByNameQueue, s.getByNameResult
}

func (s StorageUser) updateQueue(
	batch *pgx.Batch,
) {
	query := "UPDATE storage_user SET role=$2, active=$3 WHERE id=$1"
	batch.Queue(query, s.ID, s.Role.Name, s.Active)
}

func (s *StorageUser) updateResult(results pgx.BatchResults) (err error) {
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

func (s *StorageUser) Update() (BatchOperation, BatchRead) {
	return s.updateQueue, s.updateResult
}
