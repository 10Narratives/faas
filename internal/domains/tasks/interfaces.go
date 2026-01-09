package taskdomain

import (
	"context"
)

type TaskCreator interface {
	CreateTask(ctx context.Context, args *CreateTaskArgs) (*CreateTaskResult, error)
}

type CreateTaskArgs struct {
	Function   string
	Parameters string
}

type CreateTaskResult struct {
	Name string
}

type TaskPublisher interface {
	PublishExecute(ctx context.Context, msg *ExecuteTaskMessage) error
	PublishCancel(ctx context.Context, msg *CancelTaskMessage) error
}

type ExecuteTaskMessage struct {
	TaskName TaskName `json:"task_name"`
}

type CancelTaskMessage struct {
	TaskName TaskName `json:"task_name"`
}

type TaskGetter interface {
	GetTask(ctx context.Context, args *GetTaskArgs) (*GetTaskResult, error)
}

type GetTaskArgs struct {
	Name string
}

type GetTaskResult struct {
	Task *Task
}

type TaskLister interface {
	ListTasks(ctx context.Context, args *ListTasksArgs) (*ListTaskResult, error)
}

type ListTasksArgs struct {
	PageSize  int32
	PageToken string
}

type ListTaskResult struct {
	Tasks         []*Task
	NextPageToken string
}

type TaskDeleter interface {
	DeleteTask(ctx context.Context, args *DeleteTaskArgs) error
}

type DeleteTaskArgs struct {
	Name string
}

type TaskCanceler interface {
	CancelTask(ctx context.Context, args *CancelTaskArgs) (*CancelTaskResult, error)
}

type CancelTaskArgs struct {
	Name string
}

type CancelTaskResult struct {
	Task *Task
}
