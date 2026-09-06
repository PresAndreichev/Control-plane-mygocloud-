package postgres

import (
	"context"
	"fmt"
	"hash/fnv"

	"control-plane/internal/lock"

	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresLock implements lock.Locker using PostgreSQL advisory locks.
// It acquires a dedicated connection from the pool for each lock so that
// pg_advisory_lock / pg_advisory_unlock run on the same session.
type postgresLock struct {
	pool *pgxpool.Pool
}

// New creates a PostgreSQL-backed distributed locker.
func New(pool *pgxpool.Pool) lock.Locker {
	return &postgresLock{pool: pool}
}

func (l *postgresLock) Lock(ctx context.Context, key string) (lock.UnlockFunc, error) {
	id := hashKey(key)

	// Acquire a dedicated connection so lock/unlock happen on the same session.
	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire conn for lock: %w", err)
	}

	_, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", id)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("pg_advisory_lock: %w", err)
	}

	return func() error {
		defer conn.Release()
		_, err := conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", id)
		return err
	}, nil
}

func hashKey(key string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return int64(h.Sum64())
}
