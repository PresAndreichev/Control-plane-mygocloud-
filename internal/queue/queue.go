package queue

import "context"

type JobType string

const (
	JobTypeDeploy   JobType = "deploy"
	JobTypeRollback JobType = "rollback"
	JobTypeScale    JobType = "scale"
	JobTypeDestroy  JobType = "destroy"
	JobTypeK8Apply  JobType = "k8_apply"
)

type DeploymentJob struct {
	Type          JobType `json:"type"`
	DeploymentID  string  `json:"deployment_id"`
	ApplicationID string  `json:"application_id"`
	Image         string  `json:"image"`
	Version       string  `json:"version"`
}

type Queue interface {
	Publish(ctx context.Context, job DeploymentJob) error
	Consume(ctx context.Context) (<-chan DeploymentJob, error)
	Close() error
}
