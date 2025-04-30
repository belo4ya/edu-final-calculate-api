package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func Connect(ctx context.Context, path string) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, "sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("sqlx connect: %w", err)
	}
	return db, nil
}
