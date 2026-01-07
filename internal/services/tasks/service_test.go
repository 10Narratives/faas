package tasksrv_test

import (
	"context"
	"errors"
	"testing"

	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	tasksrv "github.com/10Narratives/faas/internal/services/tasks"
	"github.com/10Narratives/faas/internal/services/tasks/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CancelTask(t *testing.T) {
	t.Parallel()

	t.Run("nil args -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewTaskRepository(t)
		svc := tasksrv.NewService(repo)

		res, err := svc.CancelTask(context.Background(), nil)
		require.Nil(t, res)
		require.Error(t, err)
		require.True(t, errors.Is(err, taskdomain.ErrInvalidName))
	})

	t.Run("empty name -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewTaskRepository(t)
		svc := tasksrv.NewService(repo)

		res, err := svc.CancelTask(context.Background(), &taskdomain.CancelTaskArgs{Name: ""})
		require.Nil(t, res)
		require.Error(t, err)
		require.True(t, errors.Is(err, taskdomain.ErrInvalidName))
	})

	t.Run("bad name format -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewTaskRepository(t)
		svc := tasksrv.NewService(repo)

		res, err := svc.CancelTask(context.Background(), &taskdomain.CancelTaskArgs{Name: "bad"})
		require.Nil(t, res)
		require.Error(t, err)
		require.True(t, errors.Is(err, taskdomain.ErrInvalidName))
	})

	t.Run("repo error -> passthrough", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewTaskRepository(t)
		svc := tasksrv.NewService(repo)

		args := &taskdomain.CancelTaskArgs{Name: "tasks/1"}

		repo.EXPECT().
			CancelTask(mock.Anything, args).
			Return((*taskdomain.CancelTaskResult)(nil), taskdomain.ErrCannotCancelTask)

		res, err := svc.CancelTask(context.Background(), args)
		require.Nil(t, res)
		require.Error(t, err)
		require.True(t, errors.Is(err, taskdomain.ErrCannotCancelTask))
	})

	t.Run("repo returns nil result -> ErrNotFound", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewTaskRepository(t)
		svc := tasksrv.NewService(repo)

		args := &taskdomain.CancelTaskArgs{Name: "tasks/1"}

		repo.EXPECT().
			CancelTask(mock.Anything, args).
			Return((*taskdomain.CancelTaskResult)(nil), nil)

		res, err := svc.CancelTask(context.Background(), args)
		require.Nil(t, res)
		require.Error(t, err)
		require.True(t, errors.Is(err, taskdomain.ErrNotFound))
	})

	t.Run("repo returns result with nil task -> ErrNotFound", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewTaskRepository(t)
		svc := tasksrv.NewService(repo)

		args := &taskdomain.CancelTaskArgs{Name: "tasks/1"}

		repo.EXPECT().
			CancelTask(mock.Anything, args).
			Return(&taskdomain.CancelTaskResult{Task: nil}, nil)

		res, err := svc.CancelTask(context.Background(), args)
		require.Nil(t, res)
		require.Error(t, err)
		require.True(t, errors.Is(err, taskdomain.ErrNotFound))
	})

	t.Run("ok -> returns repo result", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewTaskRepository(t)
		svc := tasksrv.NewService(repo)

		args := &taskdomain.CancelTaskArgs{Name: "tasks/1"}

		want := &taskdomain.CancelTaskResult{
			Task: &taskdomain.Task{
				Name:       taskdomain.TaskName("tasks/1"),
				Function:   "fn",
				Parameters: "{}",
				State:      taskdomain.TaskStateCanceled,
			},
		}

		repo.EXPECT().
			CancelTask(mock.Anything, args).
			Return(want, nil)

		got, err := svc.CancelTask(context.Background(), args)
		require.NoError(t, err)
		require.Same(t, want, got)
		require.NotNil(t, got.Task)
		require.Equal(t, "tasks/1", string(got.Task.Name))
	})
}

func TestService_DeleteTask_Delegates(t *testing.T) {
	t.Parallel()

	repo := mocks.NewTaskRepository(t)
	svc := tasksrv.NewService(repo)

	args := &taskdomain.DeleteTaskArgs{Name: "tasks/1"}

	repo.EXPECT().
		DeleteTask(mock.Anything, args).
		Return(nil)

	err := svc.DeleteTask(context.Background(), args)
	require.NoError(t, err)
}

func TestService_ListTasks_Delegates(t *testing.T) {
	t.Parallel()

	repo := mocks.NewTaskRepository(t)
	svc := tasksrv.NewService(repo)

	args := &taskdomain.ListTasksArgs{PageSize: 10, PageToken: "p"}
	want := &taskdomain.ListTaskResult{
		Tasks:         []*taskdomain.Task{{Name: taskdomain.TaskName("tasks/1")}},
		NextPageToken: "next",
	}

	repo.EXPECT().
		ListTasks(mock.Anything, args).
		Return(want, nil)

	got, err := svc.ListTasks(context.Background(), args)
	require.NoError(t, err)
	require.Same(t, want, got)
	require.Equal(t, "next", got.NextPageToken)
	require.Len(t, got.Tasks, 1)
}

func TestService_GetTask_Delegates(t *testing.T) {
	t.Parallel()

	repo := mocks.NewTaskRepository(t)
	svc := tasksrv.NewService(repo)

	args := &taskdomain.GetTaskArgs{Name: "tasks/1"}
	want := &taskdomain.GetTaskResult{
		Task: &taskdomain.Task{Name: taskdomain.TaskName("tasks/1")},
	}

	repo.EXPECT().
		GetTask(mock.Anything, args).
		Return(want, nil)

	got, err := svc.GetTask(context.Background(), args)
	require.NoError(t, err)
	require.Same(t, want, got)
	require.NotNil(t, got.Task)
}

func TestService_CreateTask_Delegates(t *testing.T) {
	t.Parallel()

	repo := mocks.NewTaskRepository(t)
	svc := tasksrv.NewService(repo)

	args := &taskdomain.CreateTaskArgs{Function: "fn", Parameters: "{}"}
	want := &taskdomain.CreateTaskResult{}

	repo.EXPECT().
		CreateTask(mock.Anything, args).
		Return(want, nil)

	got, err := svc.CreateTask(context.Background(), args)
	require.NoError(t, err)
	require.Same(t, want, got)
}
