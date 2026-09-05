package queue

import "context"

type DeploymentJob struct {
	DeploymentID  string `json:"deployment_id"`
	ApplicationID string `json:"application_id"`
	Image         string `json:"image"`
	Version       string `json:"version"`
}

type Queue interface {
	Publish(ctx context.Context, job DeploymentJob) error
	Consume(ctx context.Context) (<-chan DeploymentJob, error)

	Close() error
}
