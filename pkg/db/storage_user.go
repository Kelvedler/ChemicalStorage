package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StorageUserShort struct {
	Name     string `json:"name"     validate:"gte=3,lte=50" uaLocal:"логін"`
	Password string `json:"password" validate:"gte=6,lte=20" uaLocal:"пароль"`
}

type StorageUserDBInsert struct {
	Name     string `json:"name"     validate:"gte=3,lte=50" uaLocal:"логін"`
	Password string `json:"password" validate:"gte=6,lte=20" uaLocal:"пароль"`
	Role     string `json:"role"                             uaLocal:"роль"`
}

type StorageUserFull struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Password  string    `json:"password"`
	Active    bool      `json:"active"`
}

const (
	RoleAdmin       = "admin"
	RoleAssistant   = "assistant"
	RoleLecturer    = "lecturer"
	RoleUnconfirmed = "unconfirmed"
)

func StorageUserCreate(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	newStorageUser StorageUserDBInsert,
) (createdStorageUser StorageUserFull, err error) {
	if newStorageUser.Role == "" {
		newStorageUser.Role = RoleUnconfirmed
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

func storageUserGet(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	query string,
	args ...any,
) (storageUser StorageUserFull, err error) {
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
) (storageUser StorageUserFull, err error) {
	query := "SELECT * FROM storage_user WHERE id=$1"
	return storageUserGet(ctx, dbpool, query, storageUserID)
}

func StorageUserGetByName(
	ctx context.Context,
	dbpool *pgxpool.Pool,
	storageUserName string,
) (storageUser StorageUserFull, err error) {
	query := "SELECT * FROM storage_user WHERE name=$1"
	return storageUserGet(ctx, dbpool, query, storageUserName)
}
