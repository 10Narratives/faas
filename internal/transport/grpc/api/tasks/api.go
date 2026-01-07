package taskapi

import (
	"context"
	"errors"
	"time"

	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	faaspb "github.com/10Narratives/faas/pkg/faas/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate mockery --name TaskService --output ./mocks --outpkg mocks --with-expecter --filename task_service.go
type TaskService interface {
	taskdomain.TaskGetter
	taskdomain.TaskLister
	taskdomain.TaskDeleter
	taskdomain.TaskCanceler
}

type Server struct {
	faaspb.UnimplementedTasksServer
	taskService TaskService
}

func NewServer(taskService TaskService) *Server {
	return &Server{taskService: taskService}
}

func NewRegistration(taskService TaskService) grpcsrv.ServiceRegistration {
	return func(s *grpc.Server) {
		faaspb.RegisterTasksServer(s, NewServer(taskService))
	}
}

func (s *Server) GetTask(ctx context.Context, req *faaspb.GetTaskRequest) (*faaspb.Task, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if _, err := taskdomain.ParseTaskName(req.GetName()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := s.taskService.GetTask(ctx, &taskdomain.GetTaskArgs{Name: req.GetName()})
	if err != nil {
		return nil, mapDomainErr(err)
	}
	if res == nil || res.Task == nil {
		return nil, status.Error(codes.Internal, "empty result")
	}

	return toPBTask(res.Task), nil
}

func (s *Server) ListTasks(ctx context.Context, req *faaspb.ListTasksRequest) (*faaspb.ListTasksResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	res, err := s.taskService.ListTasks(ctx, &taskdomain.ListTasksArgs{
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}
	if res == nil {
		return nil, status.Error(codes.Internal, "empty result")
	}

	out := &faaspb.ListTasksResponse{
		Tasks:         make([]*faaspb.Task, 0, len(res.Tasks)),
		NextPageToken: res.NextPageToken,
	}
	for _, t := range res.Tasks {
		if t == nil {
			continue
		}
		out.Tasks = append(out.Tasks, toPBTask(t))
	}

	return out, nil
}

func (s *Server) DeleteTask(ctx context.Context, req *faaspb.DeleteTaskRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if _, err := taskdomain.ParseTaskName(req.GetName()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.taskService.DeleteTask(ctx, &taskdomain.DeleteTaskArgs{Name: req.GetName()}); err != nil {
		return nil, mapDomainErr(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) CancelTask(ctx context.Context, req *faaspb.CancelTaskRequest) (*faaspb.Task, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if _, err := taskdomain.ParseTaskName(req.GetName()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := s.taskService.CancelTask(ctx, &taskdomain.CancelTaskArgs{Name: req.GetName()})
	if err != nil {
		return nil, mapDomainErr(err)
	}
	if res == nil || res.Task == nil {
		return nil, status.Error(codes.Internal, "empty result")
	}

	return toPBTask(res.Task), nil
}

func mapDomainErr(err error) error {
	switch {
	case errors.Is(err, taskdomain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, taskdomain.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, taskdomain.ErrInvalidName),
		errors.Is(err, taskdomain.ErrInvalidParameters),
		errors.Is(err, taskdomain.ErrInvalidFunction),
		errors.Is(err, taskdomain.ErrEmptyPageSize),
		errors.Is(err, taskdomain.ErrInvalidPageToken),
		errors.Is(err, taskdomain.ErrInvalidResult),
		errors.Is(err, taskdomain.ErrUnknownResultType):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, taskdomain.ErrInvalidState),
		errors.Is(err, taskdomain.ErrTaskNotPending),
		errors.Is(err, taskdomain.ErrTaskNotProcessing),
		errors.Is(err, taskdomain.ErrTaskAlreadyCompleted),
		errors.Is(err, taskdomain.ErrCannotCancelTask),
		errors.Is(err, taskdomain.ErrResultAlreadySet):
		return status.Error(codes.FailedPrecondition, err.Error())

	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toPBTimestampOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toPBTask(t *taskdomain.Task) *faaspb.Task {
	out := &faaspb.Task{
		Name:       string(t.Name),
		Function:   t.Function,
		Parameters: t.Parameters,
		State:      toPBState(t.State),
		CreatedAt:  toPBTimestampOrNil(t.CreatedAt),
		StartedAt:  toPBTimestampOrNil(t.StartedAt),
		EndedAt:    toPBTimestampOrNil(t.EndedAt),
	}

	if t.Result != nil {
		out.Result = toPBTaskResult(t.Result)
	}
	return out
}

func toPBState(s taskdomain.TaskState) faaspb.TaskState {
	switch s {
	case taskdomain.TaskStatePending:
		return faaspb.TaskState_TASK_STATE_PENDING
	case taskdomain.TaskStateProcessing:
		return faaspb.TaskState_TASK_STATE_PROCESSING
	case taskdomain.TaskStateSucceeded:
		return faaspb.TaskState_TASK_STATE_SUCCEEDED
	case taskdomain.TaskStateFailed:
		return faaspb.TaskState_TASK_STATE_FAILED
	case taskdomain.TaskStateCanceled:
		return faaspb.TaskState_TASK_STATE_CANCELED
	default:
		return faaspb.TaskState_TASK_STATE_UNSPECIFIED
	}
}

func toPBTaskResult(tr *taskdomain.TaskResult) *faaspb.TaskResult {
	switch tr.Type {
	case taskdomain.TaskResultInline:
		return &faaspb.TaskResult{Data: &faaspb.TaskResult_InlineResult{InlineResult: tr.InlineResult}}
	case taskdomain.TaskResultObjectKey:
		return &faaspb.TaskResult{Data: &faaspb.TaskResult_ObjectKey{ObjectKey: tr.ObjectKey}}
	case taskdomain.TaskResultError:
		return &faaspb.TaskResult{Data: &faaspb.TaskResult_ErrorMessage{ErrorMessage: tr.ErrorMessage}}
	default:
		return nil
	}
}
