package channel

import (
	"context"
	"control-plane/internal/queue"
	"errors"
	"sync"
)

type Queue struct {
	ch     chan queue.DeploymentJob
	closed bool
	mu     sync.Mutex
}

func New(bufferSize int) *Queue {
	return &Queue{
		ch: make(chan queue.DeploymentJob, bufferSize),
	}
}

func (q *Queue) Publish(ctx context.Context, job queue.DeploymentJob) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return errors.New("queue is closed")
	}
	q.mu.Unlock()

	select {
	case q.ch <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *Queue) Consume(ctx context.Context) (<-chan queue.DeploymentJob, error) {
	return q.ch, nil

}

func (q *Queue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return nil
	}
	q.closed = true

	close(q.ch)

	return nil

}
