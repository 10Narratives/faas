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
	subjectTaskExecute = "tasks.execute"
	subjectTaskCancel  = "tasks.cancel"

	streamTasks = "TASKS"
)

//go:generate mockery --name Stream --output ./mocks --outpkg mocks --with-expecter --filename stream.go
type Stream interface {
	jetstream.JetStream
}

type Publisher struct {
	js Stream
}

func NewPublisher(js jetstream.JetStream) *Publisher {
	return &Publisher{js: js}
}

// ensureTasksStream implements "variant 2":
// try to get stream handle; if not found -> create stream.
func (p *Publisher) ensureTasksStream(ctx context.Context) error {
	_, err := p.js.Stream(ctx, streamTasks)
	if err == nil {
		return nil
	}
	if !errors.Is(err, jetstream.ErrStreamNotFound) {
		return fmt.Errorf("get stream %s: %w", streamTasks, err)
	}

	_, err = p.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamTasks,
		Subjects: []string{subjectTaskExecute, subjectTaskCancel},
	})
	if err != nil {
		return fmt.Errorf("create stream %s: %w", streamTasks, err)
	}
	return nil
}

func (p *Publisher) PublishCancel(ctx context.Context, msg *taskdomain.CancelTaskMessage) error {
	if msg == nil {
		return errors.New("cancel message is nil")
	}
	if msg.TaskName == "" {
		return taskdomain.ErrInvalidName
	}

	if err := p.ensureTasksStream(ctx); err != nil {
		return err
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal cancel msg: %w", err)
	}

	_, err = p.js.Publish(ctx, subjectTaskCancel, b)
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

	if err := p.ensureTasksStream(ctx); err != nil {
		return err
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal execute msg: %w", err)
	}

	_, err = p.js.Publish(ctx, subjectTaskExecute, b)
	if err != nil {
		return fmt.Errorf("jetstream publish execute: %w", err)
	}
	return nil
}
