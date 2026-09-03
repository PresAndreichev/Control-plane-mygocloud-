package executor

import "context"

type Executor interface {
	Deploy(ctx context.Context, appID, depID, image, version string) (containerID string, err error)
	Status(ctx context.Context, containerID string) (string, error)
	Logs(ctx context.Context, containerID string, tail int) (string, error)
	Stop(ctx context.Context, containerID string) error
}
