package tasksrv_test

import (
	"context"
	"errors"
	"testing"

	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	tasksrv "github.com/10Narratives/faas/internal/services/tasks"
	"github.com/10Narratives/faas/internal/services/tasks/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestService_CreateTask(t *testing.T) {
	ctx := context.Background()

	t.Run("error: repo returns error", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		args := &taskdomain.CreateTaskArgs{Function: "fn", Parameters: "{}"}
		wantErr := errors.New("repo fail")

		repo.EXPECT().CreateTask(ctx, args).Return((*taskdomain.CreateTaskResult)(nil), wantErr).Once()

		res, err := svc.CreateTask(ctx, args)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, res)
	})

	t.Run("error: repo returns nil result", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		args := &taskdomain.CreateTaskArgs{Function: "fn", Parameters: "{}"}

		repo.EXPECT().CreateTask(ctx, args).Return((*taskdomain.CreateTaskResult)(nil), nil).Once()

		res, err := svc.CreateTask(ctx, args)
		require.ErrorIs(t, err, taskdomain.ErrAlreadyExists)
		require.Nil(t, res)
	})

	t.Run("error: repo returns empty name", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		args := &taskdomain.CreateTaskArgs{Function: "fn", Parameters: "{}"}

		repo.EXPECT().CreateTask(ctx, args).Return(&taskdomain.CreateTaskResult{Name: ""}, nil).Once()

		res, err := svc.CreateTask(ctx, args)
		require.ErrorIs(t, err, taskdomain.ErrAlreadyExists)
		require.Nil(t, res)
	})

	t.Run("ok: publish execute called; ignore publish error", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		args := &taskdomain.CreateTaskArgs{Function: "fn", Parameters: "{}"}
		repoRes := &taskdomain.CreateTaskResult{Name: "tasks/123"}

		repo.EXPECT().CreateTask(ctx, args).Return(repoRes, nil).Once()
		pub.EXPECT().
			PublishExecute(ctx, &taskdomain.ExecuteTaskMessage{TaskName: taskdomain.TaskName("tasks/123")}).
			Return(errors.New("publish failed")).
			Once()

		res, err := svc.CreateTask(ctx, args)
		require.NoError(t, err)
		require.Equal(t, repoRes, res)
	})
}

func TestService_CancelTask(t *testing.T) {
	ctx := context.Background()

	t.Run("error: args nil", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		res, err := svc.CancelTask(ctx, nil)
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
		require.Nil(t, res)
	})

	t.Run("error: empty name", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		res, err := svc.CancelTask(ctx, &taskdomain.CancelTaskArgs{Name: ""})
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
		require.Nil(t, res)
	})

	t.Run("error: invalid name format", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		res, err := svc.CancelTask(ctx, &taskdomain.CancelTaskArgs{Name: "bad/123"})
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
		require.Nil(t, res)
	})

	t.Run("error: repo returns error", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		args := &taskdomain.CancelTaskArgs{Name: "tasks/123"}
		wantErr := errors.New("repo fail")

		repo.EXPECT().CancelTask(ctx, args).Return((*taskdomain.CancelTaskResult)(nil), wantErr).Once()

		res, err := svc.CancelTask(ctx, args)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, res)
	})

	t.Run("error: repo returns nil result", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		args := &taskdomain.CancelTaskArgs{Name: "tasks/123"}

		repo.EXPECT().CancelTask(ctx, args).Return((*taskdomain.CancelTaskResult)(nil), nil).Once()

		res, err := svc.CancelTask(ctx, args)
		require.ErrorIs(t, err, taskdomain.ErrNotFound)
		require.Nil(t, res)
	})

	t.Run("error: repo returns result with nil task", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		args := &taskdomain.CancelTaskArgs{Name: "tasks/123"}

		repo.EXPECT().CancelTask(ctx, args).Return(&taskdomain.CancelTaskResult{Task: nil}, nil).Once()

		res, err := svc.CancelTask(ctx, args)
		require.ErrorIs(t, err, taskdomain.ErrNotFound)
		require.Nil(t, res)
	})

	t.Run("ok: publish cancel called; ignore publish error", func(t *testing.T) {
		repo := mocks.NewTaskRepository(t)
		pub := mocks.NewTaskPublisher(t)

		svc := tasksrv.NewService(repo, pub)

		args := &taskdomain.CancelTaskArgs{Name: "tasks/123"}

		task := &taskdomain.Task{
			ID:   uuid.New(),
			Name: taskdomain.TaskName("tasks/123"),
		}
		repoRes := &taskdomain.CancelTaskResult{Task: task}

		repo.EXPECT().CancelTask(ctx, args).Return(repoRes, nil).Once()
		pub.EXPECT().
			PublishCancel(ctx, &taskdomain.CancelTaskMessage{TaskName: taskdomain.TaskName("tasks/123")}).
			Return(errors.New("publish failed")).
			Once()

		res, err := svc.CancelTask(ctx, args)
		require.NoError(t, err)
		require.Equal(t, repoRes, res)
	})
}
