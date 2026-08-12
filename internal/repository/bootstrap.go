package repository

import (
	"context"
	"errors"

	db "github.com/allifiz/go-opname-api/internal/database/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInitialUserAlreadyExists = errors.New("initial user already exists")
	ErrRoleNotFound             = errors.New("role not found")
)

func (s *Store) BootstrapInitialUser(ctx context.Context, name, email, passwordHash, roleCode string) (db.User, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return db.User{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return db.User{}, err
	}

	var userCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return db.User{}, err
	}
	if userCount != 0 {
		return db.User{}, ErrInitialUserAlreadyExists
	}

	var roleID pgtype.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE code = $1`, roleCode).Scan(&roleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, ErrRoleNotFound
		}
		return db.User{}, err
	}

	var user db.User
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (role_id, name, email, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, role_id, name, email, password_hash, is_active, created_at, updated_at`,
		roleID, name, email, passwordHash,
	).Scan(
		&user.ID,
		&user.RoleID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return db.User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.User{}, err
	}
	return user, nil
}
