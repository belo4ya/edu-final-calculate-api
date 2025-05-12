package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/repository/models"

	"github.com/mattn/go-sqlite3"
	"github.com/rs/xid"
)

// Register creates a new user account with the provided credentials.
// Returns [models.ErrUserExists] if a user with the same login already exists.
func (r *Repository) Register(ctx context.Context, cmd models.RegisterUserCmd) error {
	const q = `INSERT INTO users (id, login, password_hash) VALUES (?, ?, ?)`

	_, err := r.db.ExecContext(ctx, q, xid.New().String(), cmd.Login, cmd.PasswordHash)
	if err == nil {
		return nil
	}

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if errors.Is(sqliteErr.Code, sqlite3.ErrConstraint) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			return models.ErrUserExists
		}
	}

	return fmt.Errorf("db exec: %w", err)
}

// GetUser retrieves a user by login and password hash.
// Returns [models.ErrUserNotFound] if no matching user exists.
func (r *Repository) GetUser(ctx context.Context, cmd models.GetUserCmd) (*models.User, error) {
	const q = `
		SELECT id, login, password_hash, created_at, updated_at
		FROM users
		WHERE login = ? AND password_hash = ?
		`

	var user models.User
	if err := r.db.GetContext(ctx, &user, q, cmd.Login, cmd.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrUserNotFound
		}
		return nil, fmt.Errorf("db get: %w", err)
	}

	return &user, nil
}
