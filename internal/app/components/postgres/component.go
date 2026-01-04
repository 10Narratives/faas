package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnection(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
