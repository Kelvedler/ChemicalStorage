package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleName string

const (
	Admin       RoleName = "admin"
	Assistant   RoleName = "assistant"
	Lecturer    RoleName = "lecturer"
	Unconfirmed RoleName = "unconfirmed"
)

func IsValidRole(role string) bool {
	roleNames := []RoleName{Admin, Assistant, Lecturer, Unconfirmed}
	for _, roleName := range roleNames {
		if role == string(roleName) {
			return true
		}
	}
	return false
}

var RoleInvalid = errors.New("Storage user role is not valid")

type StorageUserInput struct {
	ID        string
	CreatedAt string
	UpdatedAt string
	Name      string
	Role      string
	Password  string
	Active    string
}

type StorageUser struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"       validate:"gte=3,lte=50" uaLocal:"логін"`
	Role      RoleName  `json:"role"                               uaLocal:"роль"`
	Password  string    `json:"password"   validate:"gte=6,lte=20" uaLocal:"пароль"`
	Active    bool      `json:"active"`
}

func (input StorageUserInput) StorageUserBind() (output StorageUser, err error) {
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
		if !IsValidRole(input.Role) {
			return StorageUser{}, nil
		} else {
			output.Role = RoleName(input.Role)
		}
	}
	output.Password = input.Password
	if input.Active == "" {
		input.Active = "false"
	}
	active, err := strconv.ParseBool(input.Active)
	if err != nil {
		return StorageUser{}, err
	}
	output.Active = active

	return output, nil
}

func StorageUserCreate(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	newStorageUser StorageUser,
) (createdStorageUser StorageUser, err error) {
	if newStorageUser.Role == "" {
		newStorageUser.Role = Unconfirmed
	}
	query := "INSERT into storage_user(name, password, role) VALUES($1, $2, $3) RETURNING id, created_at, updated_at, name, role, password, active"
	err = dbpool.QueryRow(
		ctx,
		query,
		newStorageUser.Name,
		newStorageUser.Password,
		newStorageUser.Role,
	).Scan(
		&createdStorageUser.ID,
		&createdStorageUser.CreatedAt,
		&createdStorageUser.UpdatedAt,
		&createdStorageUser.Name,
		&createdStorageUser.Role,
		&createdStorageUser.Password,
		&createdStorageUser.Active,
	)
	return createdStorageUser, err
}

func storageUserGetSlice(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	query string,
	args ...any,
) ([]StorageUser, error) {
	storageUsersSlice := make([]StorageUser, 0)
	rows, err := dbpool.Query(ctx, query, args...)
	if err != nil {
		return storageUsersSlice, err
	}
	next := rows.Next()
	if !next {
		return storageUsersSlice, nil
	}
	for next {
		var storageUser StorageUser
		rows.Scan(
			&storageUser.ID,
			&storageUser.CreatedAt,
			&storageUser.UpdatedAt,
			&storageUser.Name,
			&storageUser.Role,
			&storageUser.Password,
			&storageUser.Active,
		)
		storageUsersSlice = append(storageUsersSlice, storageUser)
		next = rows.Next()
	}
	return storageUsersSlice, nil
}

func StorageUserGetRange(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	limit int,
	offset int,
	src string,
	excludeID string,
) ([]StorageUser, error) {
	orderBy := "-created_at"
	if len(src) >= 1 {
		query := "SELECT * FROM storage_user WHERE name ILIKE $4 AND id!=$5 ORDER BY $1 LIMIT $2 OFFSET $3"
		return storageUserGetSlice(ctx, dbpool, query, orderBy, limit, offset, src+"%", excludeID)
	} else {
		query := "SELECT * FROM storage_user WHERE id!=$4 ORDER BY $1 LIMIT $2 OFFSET $3"
		return storageUserGetSlice(ctx, dbpool, query, orderBy, limit, offset, excludeID)
	}
}

func storageUserGet(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	query string,
	args ...any,
) (storageUser StorageUser, err error) {
	err = dbpool.QueryRow(ctx, query, args...).Scan(
		&storageUser.ID,
		&storageUser.CreatedAt,
		&storageUser.UpdatedAt,
		&storageUser.Name,
		&storageUser.Role,
		&storageUser.Password,
		&storageUser.Active,
	)
	return storageUser, err
}

func StorageUserGetByID(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	storageUserID string,
) (storageUser StorageUser, err error) {
	query := "SELECT * FROM storage_user WHERE id=$1"
	return storageUserGet(ctx, dbpool, query, storageUserID)
}

func StorageUserGetByName(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	storageUserName string,
) (storageUser StorageUser, err error) {
	query := "SELECT * FROM storage_user WHERE name=$1"
	return storageUserGet(ctx, dbpool, query, storageUserName)
}

func StorageUserUpdate(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	updateData StorageUser,
) error {
	query := "UPDATE storage_user SET role=$2, active=$3 WHERE id=$1"
	result, err := dbpool.Exec(ctx, query, updateData.ID, updateData.Role, updateData.Active)
	affectedRows := result.RowsAffected()
	if err != nil {
		return err
	} else if affectedRows != 1 {
		if affectedRows == 0 {
			return pgx.ErrNoRows
		} else {
			panic(fmt.Sprintf("Update affected more than one row - %d", affectedRows))
		}
	}
	return nil
}
