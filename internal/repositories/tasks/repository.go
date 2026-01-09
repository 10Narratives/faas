package taskrepo

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

//go:generate mockery --name KeyValue --output ./mocks --outpkg mocks --with-expecter --filename key_value.go
type KeyValue interface {
	jetstream.KeyValue
}

type Repository struct {
	kv KeyValue
}

func NewRepository(ctx context.Context, js jetstream.JetStream, bucket string) (*Repository, error) {
	if bucket == "" {
		return nil, errors.New("bucket is empty")
	}

	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: bucket,
	})
	if err != nil {
		return nil, err
	}

	return &Repository{kv: kv}, nil
}

func (r *Repository) CancelTask(ctx context.Context, args *taskdomain.CancelTaskArgs) (*taskdomain.CancelTaskResult, error) {
	if args == nil || args.Name == "" {
		return nil, taskdomain.ErrInvalidName
	}
	if _, err := taskdomain.ParseTaskName(args.Name); err != nil {
		return nil, err
	}

	entry, t, err := r.getTaskEntry(ctx, args.Name)
	if err != nil {
		return nil, err
	}

	switch t.State {
	case taskdomain.TaskStatePending, taskdomain.TaskStateProcessing:
		// ok
	case taskdomain.TaskStateSucceeded, taskdomain.TaskStateFailed, taskdomain.TaskStateCanceled:
		return nil, taskdomain.ErrTaskAlreadyCompleted
	default:
		return nil, taskdomain.ErrInvalidState
	}

	t.State = taskdomain.TaskStateCanceled
	t.EndedAt = time.Now().UTC()

	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	if _, err := r.kv.Update(ctx, args.Name, b, entry.Revision()); err != nil {
		return nil, taskdomain.ErrCannotCancelTask
	}

	return &taskdomain.CancelTaskResult{Task: t}, nil
}

func (r *Repository) CreateTask(ctx context.Context, args *taskdomain.CreateTaskArgs) (*taskdomain.CreateTaskResult, error) {
	if args == nil {
		return nil, taskdomain.ErrInvalidParameters
	}
	if strings.TrimSpace(args.Function) == "" {
		return nil, taskdomain.ErrInvalidFunction
	}

	id := uuid.New()
	name := "tasks/" + id.String()
	now := time.Now().UTC()

	t := &taskdomain.Task{
		ID:         id,
		Name:       taskdomain.TaskName(name),
		Function:   args.Function,
		Parameters: args.Parameters,
		State:      taskdomain.TaskStatePending,
		CreatedAt:  now,
	}

	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	_, err = r.kv.Create(ctx, name, b)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyExists) {
			return nil, taskdomain.ErrAlreadyExists
		}
		return nil, err
	}

	return &taskdomain.CreateTaskResult{Name: name}, nil
}

func (r *Repository) DeleteTask(ctx context.Context, args *taskdomain.DeleteTaskArgs) error {
	if args == nil || args.Name == "" {
		return taskdomain.ErrInvalidName
	}
	if _, err := taskdomain.ParseTaskName(args.Name); err != nil {
		return err
	}

	if err := r.kv.Delete(ctx, args.Name); err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return taskdomain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *Repository) GetTask(ctx context.Context, args *taskdomain.GetTaskArgs) (*taskdomain.GetTaskResult, error) {
	if args == nil || args.Name == "" {
		return nil, taskdomain.ErrInvalidName
	}
	if _, err := taskdomain.ParseTaskName(args.Name); err != nil {
		return nil, err
	}

	_, t, err := r.getTaskEntry(ctx, args.Name)
	if err != nil {
		return nil, err
	}

	return &taskdomain.GetTaskResult{Task: t}, nil
}

func (r *Repository) ListTasks(ctx context.Context, args *taskdomain.ListTasksArgs) (*taskdomain.ListTaskResult, error) {
	if args == nil {
		return nil, taskdomain.ErrInvalidParameters
	}
	if args.PageSize <= 0 {
		return nil, taskdomain.ErrEmptyPageSize
	}

	keys, err := r.listAllTaskKeys(ctx)
	if err != nil {
		return nil, err
	}

	start := 0
	if args.PageToken != "" {
		// page_token = последний key из предыдущей страницы
		i := indexOf(keys, args.PageToken)
		if i < 0 {
			return nil, taskdomain.ErrInvalidPageToken
		}
		start = i + 1
	}

	if start >= len(keys) {
		return &taskdomain.ListTaskResult{Tasks: []*taskdomain.Task{}, NextPageToken: ""}, nil
	}

	end := start + int(args.PageSize)
	if end > len(keys) {
		end = len(keys)
	}

	pageKeys := keys[start:end]
	tasks := make([]*taskdomain.Task, 0, len(pageKeys))

	for _, k := range pageKeys {
		_, t, err := r.getTaskEntry(ctx, k)
		if err != nil {
			// если ключ внезапно пропал между ListKeys и Get — просто пропускаем
			if errors.Is(err, taskdomain.ErrNotFound) {
				continue
			}
			return nil, err
		}
		tasks = append(tasks, t)
	}

	next := ""
	if end < len(keys) && len(pageKeys) > 0 {
		next = pageKeys[len(pageKeys)-1]
	}

	return &taskdomain.ListTaskResult{
		Tasks:         tasks,
		NextPageToken: next,
	}, nil
}

func (r *Repository) getTaskEntry(ctx context.Context, key string) (jetstream.KeyValueEntry, *taskdomain.Task, error) {
	entry, err := r.kv.Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, nil, taskdomain.ErrNotFound
		}
		return nil, nil, err
	}

	var t taskdomain.Task
	if err := json.Unmarshal(entry.Value(), &t); err != nil {
		return nil, nil, err
	}
	if t.Name == "" {
		t.Name = taskdomain.TaskName(key)
	}

	return entry, &t, nil
}

func (r *Repository) listAllTaskKeys(ctx context.Context) ([]string, error) {
	lister, err := r.kv.ListKeys(ctx)
	if err != nil {
		return nil, err
	}
	defer lister.Stop()

	var keys []string
	for k := range lister.Keys() {
		// в бакете могут быть другие записи; оставим только задачи
		if strings.HasPrefix(k, "tasks/") {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

func indexOf(ss []string, s string) int {
	for i := range ss {
		if ss[i] == s {
			return i
		}
	}
	return -1
}
