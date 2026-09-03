package postgres

import (
	"context"
	"control-plane/internal/domain"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type deploymentRepo struct {
	pool *pgxpool.Pool
}

func NewDeploymentRepository(pool *pgxpool.Pool) domain.DeploymentRepository {
	return &deploymentRepo{pool: pool}
}

func (r *deploymentRepo) Create(ctx context.Context, dep *domain.Deployment) error {
	query := `
		INSERT INTO deployments (id, application_id, version, status, message, container_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query, dep.ID, dep.ApplicationID, dep.Version, dep.Status, dep.Message, dep.ContainerID, dep.CreatedAt, dep.UpdatedAt)
	return err
}

func (r *deploymentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deployment, error) {
	query := `
		SELECT id, application_id, version, status, message, container_id, created_at, updated_at, completed_at
		FROM deployments WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)

	var d domain.Deployment
	var comletedAt *time.Time
	err := row.Scan(&d.ID, &d.ApplicationID, &d.Version, &d.Status, &d.Message, &d.ContainerID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	d.CompletedAt = comletedAt
	return &d, nil
}

func (r *deploymentRepo) ListByApplication(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	query := `
		SELECT id, application_id, version, status, message, container_id, created_at, updated_at, completed_at
		FROM deployments WHERE application_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []domain.Deployment
	for rows.Next() {
		var d domain.Deployment
		var completedAt *time.Time
		if err := rows.Scan(&d.ID, &d.ApplicationID, &d.Version, &d.Status, &d.Message, &d.ContainerID, &d.CreatedAt, &d.UpdatedAt, &d.CompletedAt); err != nil {
			return nil, err
		}
		d.CompletedAt = completedAt
		deps = append(deps, d)
	}
	return deps, rows.Err()

}

func (r *deploymentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status, message string) error {
	query := `
		UPDATE deployments
		SET status = $2, 
		    message = $3, 
		    updated_at = $4, 
		    completed_at = CASE 
		        WHEN $2::varchar IN ('successful', 'failed', 'rolled_back') THEN $4 
		        ELSE completed_at 
		    END
		WHERE id = $1
	`

	now := time.Now()
	_, err := r.pool.Exec(ctx, query, id, status, message, now)
	return err
}

func (r *deploymentRepo) GetLastSuccessful(ctx context.Context, appID uuid.UUID) (*domain.Deployment, error) {
	query := `
		SELECT id, application_id, version, status, message, created_at, updated_at, completed_at
		FROM deployments
		WHERE application_id = $1 AND status = 'successful'
		ORDER BY created_at DESC LIMIT 1
	`

	row := r.pool.QueryRow(ctx, query, appID)

	var d domain.Deployment
	var completedAt *time.Time

	err := row.Scan(&d.ID, &d.ApplicationID, &d.Version, &d.Status, &d.Message, &d.CreatedAt, &d.UpdatedAt, &completedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	d.CompletedAt = completedAt
	return &d, nil

}

func (r *deploymentRepo) UpdateContainerID(ctx context.Context, id uuid.UUID, containerID string) error {
	query := `
		UPDATE deployments
		SET container_id = $2, updated_at = $3
		WHERE id = $1
		`
	_, err := r.pool.Exec(ctx, query, id, containerID, time.Now())
	return err
}
