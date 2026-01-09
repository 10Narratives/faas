package tasksrv

import (
	"context"

	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
)

//go:generate mockery --name TaskRepository --output ./mocks --outpkg mocks --with-expecter --filename task_repository.go
type TaskRepository interface {
	taskdomain.TaskCreator
	taskdomain.TaskGetter
	taskdomain.TaskLister
	taskdomain.TaskDeleter
	taskdomain.TaskCanceler
}

//go:generate mockery --name TaskPublisher --output ./mocks --outpkg mocks --with-expecter --filename task_publisher.go
type TaskPublisher interface {
	taskdomain.TaskPublisher
}

type Service struct {
	taskRepo TaskRepository
	taskPub  TaskPublisher
}

func NewService(
	taskRepo TaskRepository,
	taskPub TaskPublisher,
) *Service {
	return &Service{
		taskRepo: taskRepo,
		taskPub:  taskPub,
	}
}

func (s *Service) CancelTask(ctx context.Context, args *taskdomain.CancelTaskArgs) (*taskdomain.CancelTaskResult, error) {
	if args == nil || args.Name == "" {
		return nil, taskdomain.ErrInvalidName
	}
	if _, err := taskdomain.ParseTaskName(args.Name); err != nil {
		return nil, err
	}

	res, err := s.taskRepo.CancelTask(ctx, args)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Task == nil {
		return nil, taskdomain.ErrNotFound
	}

	_ = s.taskPub.PublishCancel(ctx, &taskdomain.CancelTaskMessage{
		TaskName: res.Task.Name,
	})

	return res, nil
}

func (s *Service) DeleteTask(ctx context.Context, args *taskdomain.DeleteTaskArgs) error {
	return s.taskRepo.DeleteTask(ctx, args)
}

func (s *Service) ListTasks(ctx context.Context, args *taskdomain.ListTasksArgs) (*taskdomain.ListTaskResult, error) {
	return s.taskRepo.ListTasks(ctx, args)
}

func (s *Service) GetTask(ctx context.Context, args *taskdomain.GetTaskArgs) (*taskdomain.GetTaskResult, error) {
	return s.taskRepo.GetTask(ctx, args)
}

func (s *Service) CreateTask(ctx context.Context, args *taskdomain.CreateTaskArgs) (*taskdomain.CreateTaskResult, error) {
	res, err := s.taskRepo.CreateTask(ctx, args)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Name == "" {
		return nil, taskdomain.ErrAlreadyExists
	}

	_ = s.taskPub.PublishExecute(ctx, &taskdomain.ExecuteTaskMessage{
		TaskName: taskdomain.TaskName(res.Name),
	})

	return res, nil
}
