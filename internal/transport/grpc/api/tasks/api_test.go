package taskapi_test

import (
	"context"
	"testing"
	"time"

	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	taskapi "github.com/10Narratives/faas/internal/transport/grpc/api/tasks"
	"github.com/10Narratives/faas/internal/transport/grpc/api/tasks/mocks"
	faaspb "github.com/10Narratives/faas/pkg/faas/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServer_GetTask(t *testing.T) {
	t.Parallel()

	t.Run("nil request -> InvalidArgument", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		got, err := srv.GetTask(context.Background(), nil)
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid name -> InvalidArgument", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		got, err := srv.GetTask(context.Background(), &faaspb.GetTaskRequest{Name: "bad"})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("service not found -> NotFound", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		name := "tasks/abc"
		svc.EXPECT().
			GetTask(mock.Anything, mock.MatchedBy(func(a *taskdomain.GetTaskArgs) bool {
				return a != nil && a.Name == name
			})).
			Return((*taskdomain.GetTaskResult)(nil), taskdomain.ErrNotFound)

		got, err := srv.GetTask(context.Background(), &faaspb.GetTaskRequest{Name: name})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("service returns nil result -> Internal", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		name := "tasks/abc"
		svc.EXPECT().
			GetTask(mock.Anything, mock.Anything).
			Return((*taskdomain.GetTaskResult)(nil), nil)

		got, err := srv.GetTask(context.Background(), &faaspb.GetTaskRequest{Name: name})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, st.Code())
	})

	t.Run("ok -> maps task fields + inline result", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		now := time.Now().UTC().Truncate(time.Second)
		name := "tasks/123"
		dt := &taskdomain.Task{
			Name:       taskdomain.TaskName(name),
			Function:   "fn",
			Parameters: `{"a":1}`,
			State:      taskdomain.TaskStatePending,
			CreatedAt:  now,
			StartedAt:  time.Time{},
			EndedAt:    time.Time{},
			Result: &taskdomain.TaskResult{
				Type:         taskdomain.TaskResultInline,
				InlineResult: []byte("ok"),
			},
		}

		svc.EXPECT().
			GetTask(mock.Anything, mock.MatchedBy(func(a *taskdomain.GetTaskArgs) bool {
				return a != nil && a.Name == name
			})).
			Return(&taskdomain.GetTaskResult{Task: dt}, nil)

		got, err := srv.GetTask(context.Background(), &faaspb.GetTaskRequest{Name: name})
		require.NoError(t, err)
		require.NotNil(t, got)

		require.Equal(t, name, got.GetName())
		require.Equal(t, "fn", got.GetFunction())
		require.Equal(t, `{"a":1}`, got.GetParameters())
		require.Equal(t, faaspb.TaskState_TASK_STATE_PENDING, got.GetState())

		require.NotNil(t, got.GetCreatedAt())
		require.True(t, got.GetCreatedAt().AsTime().Equal(now))

		// started_at/ended_at должны быть nil для zero time
		require.Nil(t, got.GetStartedAt())
		require.Nil(t, got.GetEndedAt())

		require.NotNil(t, got.GetResult())
		_, ok := got.GetResult().GetData().(*faaspb.TaskResult_InlineResult)
		require.True(t, ok)
		require.Equal(t, []byte("ok"), got.GetResult().GetInlineResult())
	})
}

func TestServer_ListTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil request -> InvalidArgument", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		got, err := srv.ListTasks(context.Background(), nil)
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("service invalid page size -> InvalidArgument", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		svc.EXPECT().
			ListTasks(mock.Anything, mock.Anything).
			Return((*taskdomain.ListTaskResult)(nil), taskdomain.ErrEmptyPageSize)

		got, err := srv.ListTasks(context.Background(), &faaspb.ListTasksRequest{PageSize: 0})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("service returns nil result -> Internal", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		svc.EXPECT().
			ListTasks(mock.Anything, mock.Anything).
			Return((*taskdomain.ListTaskResult)(nil), nil)

		got, err := srv.ListTasks(context.Background(), &faaspb.ListTasksRequest{PageSize: 10})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, st.Code())
	})

	t.Run("ok -> maps tasks and skips nil", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		t1 := &taskdomain.Task{
			Name:       taskdomain.TaskName("tasks/1"),
			Function:   "f1",
			Parameters: "{}",
			State:      taskdomain.TaskStateSucceeded,
		}
		var tnil *taskdomain.Task = nil
		t2 := &taskdomain.Task{
			Name:       taskdomain.TaskName("tasks/2"),
			Function:   "f2",
			Parameters: "{}",
			State:      taskdomain.TaskStateFailed,
			Result: &taskdomain.TaskResult{
				Type:         taskdomain.TaskResultError,
				ErrorMessage: "boom",
			},
		}

		svc.EXPECT().
			ListTasks(mock.Anything, mock.MatchedBy(func(a *taskdomain.ListTasksArgs) bool {
				return a != nil && a.PageSize == 2 && a.PageToken == "p"
			})).
			Return(&taskdomain.ListTaskResult{
				Tasks:         []*taskdomain.Task{t1, tnil, t2},
				NextPageToken: "next",
			}, nil)

		got, err := srv.ListTasks(context.Background(), &faaspb.ListTasksRequest{
			PageSize:  2,
			PageToken: "p",
		})
		require.NoError(t, err)
		require.NotNil(t, got)

		require.Equal(t, "next", got.GetNextPageToken())
		require.Len(t, got.GetTasks(), 2)

		require.Equal(t, "tasks/1", got.GetTasks()[0].GetName())
		require.Equal(t, faaspb.TaskState_TASK_STATE_SUCCEEDED, got.GetTasks()[0].GetState())

		require.Equal(t, "tasks/2", got.GetTasks()[1].GetName())
		require.Equal(t, faaspb.TaskState_TASK_STATE_FAILED, got.GetTasks()[1].GetState())
		require.NotNil(t, got.GetTasks()[1].GetResult())
		require.Equal(t, "boom", got.GetTasks()[1].GetResult().GetErrorMessage())
	})
}

func TestServer_DeleteTask(t *testing.T) {
	t.Parallel()

	t.Run("nil request -> InvalidArgument", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		got, err := srv.DeleteTask(context.Background(), nil)
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid name -> InvalidArgument", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		got, err := srv.DeleteTask(context.Background(), &faaspb.DeleteTaskRequest{Name: "bad"})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("service not found -> NotFound", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		name := "tasks/404"
		svc.EXPECT().
			DeleteTask(mock.Anything, mock.MatchedBy(func(a *taskdomain.DeleteTaskArgs) bool {
				return a != nil && a.Name == name
			})).
			Return(taskdomain.ErrNotFound)

		got, err := srv.DeleteTask(context.Background(), &faaspb.DeleteTaskRequest{Name: name})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("ok -> returns empty", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		name := "tasks/1"
		svc.EXPECT().
			DeleteTask(mock.Anything, mock.Anything).
			Return(nil)

		got, err := srv.DeleteTask(context.Background(), &faaspb.DeleteTaskRequest{Name: name})
		require.NoError(t, err)
		require.NotNil(t, got)
	})
}

func TestServer_CancelTask(t *testing.T) {
	t.Parallel()

	t.Run("nil request -> InvalidArgument", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		got, err := srv.CancelTask(context.Background(), nil)
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid name -> InvalidArgument", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		got, err := srv.CancelTask(context.Background(), &faaspb.CancelTaskRequest{Name: "bad"})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("cannot cancel -> FailedPrecondition", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		name := "tasks/1"
		svc.EXPECT().
			CancelTask(mock.Anything, mock.Anything).
			Return((*taskdomain.CancelTaskResult)(nil), taskdomain.ErrCannotCancelTask)

		got, err := srv.CancelTask(context.Background(), &faaspb.CancelTaskRequest{Name: name})
		require.Nil(t, got)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.FailedPrecondition, st.Code())
	})

	t.Run("ok -> returns mapped task", func(t *testing.T) {
		t.Parallel()

		svc := mocks.NewTaskService(t)
		srv := taskapi.NewServer(svc)

		name := "tasks/77"
		dt := &taskdomain.Task{
			Name:       taskdomain.TaskName(name),
			Function:   "fn",
			Parameters: "{}",
			State:      taskdomain.TaskStateCanceled,
		}

		svc.EXPECT().
			CancelTask(mock.Anything, mock.MatchedBy(func(a *taskdomain.CancelTaskArgs) bool {
				return a != nil && a.Name == name
			})).
			Return(&taskdomain.CancelTaskResult{Task: dt}, nil)

		got, err := srv.CancelTask(context.Background(), &faaspb.CancelTaskRequest{Name: name})
		require.NoError(t, err)
		require.NotNil(t, got)

		require.Equal(t, name, got.GetName())
		require.Equal(t, faaspb.TaskState_TASK_STATE_CANCELED, got.GetState())
	})
}
