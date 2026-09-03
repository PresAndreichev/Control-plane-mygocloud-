package postgres

import (
	"context"
	"control-plane/internal/domain"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type applicationRepo struct {
	pool *pgxpool.Pool
}

func NewApplicationRepository(pool *pgxpool.Pool) domain.ApplicationRepository {
	return &applicationRepo{pool: pool}
}

func (r *applicationRepo) Create(ctx context.Context, app *domain.Application) error {
	query := `
		INSERT INTO applications (id, name, description, docker_image, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query, app.ID, app.Name, app.Description, app.DockerImage, app.OwnerID, app.CreatedAt, app.UpdatedAt)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case "23503":
				return fmt.Errorf("%w: owner does not exist", domain.ErrInvalidInput)
			case "23505":
				return domain.ErrConflict
			}
		}
		return fmt.Errorf("create application: %w", err)
	}
	return nil
}

func (r *applicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	query := `
		SELECT id, name, description, COALESCE(docker_image, ''), owner_id, created_at, updated_at 
		FROM applications WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	var a domain.Application

	err := row.Scan(&a.ID, &a.Name, &a.Description, &a.DockerImage, &a.OwnerID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *applicationRepo) List(ctx context.Context) ([]domain.Application, error) {
	query := `
		SELECT id, name, description, COALESCE(docker_image, ''), owner_id, created_at, updated_at 
		FROM applications ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.Application
	for rows.Next() {
		var a domain.Application
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.DockerImage, &a.OwnerID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (r *applicationRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Application, error) {
	query := `
		SELECT id, name, description, COALESCE(docker_image, ''), owner_id, created_at, updated_at 
		FROM applications WHERE owner_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.Application
	for rows.Next() {
		var a domain.Application
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.DockerImage, &a.OwnerID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}
