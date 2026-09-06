package noop

import (
	"context"
	"errors"
)

// Executor is a no-op implementation of executor.Executor for the API server.
// It allows the API to satisfy the DeploymentService interface without
// requiring a local Docker daemon.
type Executor struct{}

// New returns a no-op executor.
func New() *Executor {
	return &Executor{}
}

func (e *Executor) Deploy(ctx context.Context, appID, depID, image, version string) (string, error) {
	return "", errors.New("noop executor: deploy not available on API server")
}

func (e *Executor) Status(ctx context.Context, containerID string) (string, error) {
	return "", errors.New("noop executor: status not available on API server")
}

func (e *Executor) Logs(ctx context.Context, containerID string, tail int) (string, error) {
	return "", errors.New("noop executor: logs not available on API server, query a worker node")
}

func (e *Executor) Stop(ctx context.Context, containerID string) error {
	return errors.New("noop executor: stop not available on API server")
}
