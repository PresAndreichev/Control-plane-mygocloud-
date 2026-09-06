package rabbitmq

import (
	"context"
	"control-plane/internal/queue"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Queue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	name    string
	closed  bool
	mu      sync.Mutex
}

func New(url, queueName string) (*Queue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", &err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("amqp channel: %w", &err)
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("queue declare: %w", &err)
	}
	return &Queue{
		conn:    conn,
		channel: ch,
		name:    queueName,
	}, nil
}

func (q *Queue) Publish(ctx context.Context, job queue.DeploymentJob) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return errors.New("queue is closed")
	}
	q.mu.Unlock()

	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	return q.channel.PublishWithContext(ctx,
		"",
		q.name,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

func (q *Queue) Consume(ctx context.Context) (<-chan queue.DeploymentJob, error) {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return nil, errors.New("queue is closed")
	}
	q.mu.Unlock()

	msgs, err := q.channel.ConsumeWithContext(ctx,
		q.name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}

	out := make(chan queue.DeploymentJob)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return

				}
				var job queue.DeploymentJob
				if err := json.Unmarshal(msg.Body, &job); err != nil {
					_ = msg.Nack(false, false)
					continue
				}
				select {
				case out <- job:
					_ = msg.Ack(false)
				case <-ctx.Done():
					_ = msg.Nack(false, true)
					return
				}
			}

		}
	}()
	return out, nil
}

func (q *Queue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	var err error

	if q.channel != nil {
		err = q.channel.Close()
	}
	if q.conn != nil {
		if cerr := q.conn.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}
