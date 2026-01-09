package taskrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	// ВАЖНО: эти subjects должны попадать под subjects стрима TASKS (например "task.*")
	subjectTaskExecute = "task.execute"
	subjectTaskCancel  = "task.cancel"

	streamTasks = "TASKS"
)

// Интерфейс под мок: publish через JetStream + ожидание конкретного стрима
type JS interface {
	Publish(ctx context.Context, subj string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error)
}

type Publisher struct {
	js JS
}

func NewPublisher(js JS) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) PublishCancel(ctx context.Context, msg *taskdomain.CancelTaskMessage) error {
	if msg == nil {
		return errors.New("cancel message is nil")
	}
	if msg.TaskName == "" {
		return taskdomain.ErrInvalidName
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal cancel msg: %w", err)
	}

	_, err = p.js.Publish(ctx, subjectTaskCancel, b, jetstream.WithExpectStream(streamTasks))
	if err != nil {
		return fmt.Errorf("jetstream publish cancel: %w", err)
	}
	return nil
}

func (p *Publisher) PublishExecute(ctx context.Context, msg *taskdomain.ExecuteTaskMessage) error {
	if msg == nil {
		return errors.New("execute message is nil")
	}
	if msg.TaskName == "" {
		return taskdomain.ErrInvalidName
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal execute msg: %w", err)
	}

	_, err = p.js.Publish(ctx, subjectTaskExecute, b, jetstream.WithExpectStream(streamTasks))
	if err != nil {
		return fmt.Errorf("jetstream publish execute: %w", err)
	}
	return nil
}
