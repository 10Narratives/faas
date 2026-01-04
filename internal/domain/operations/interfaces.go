package opdomain

import (
	"context"
	"time"
)

type OperationCanceler interface {
	CancelOperation(ctx context.Context, opts *CancelOperationOptions) error
}

type CancelOperationOptions struct {
	Name string
}

type OperationCreator interface {
	CreateOperation(ctx context.Context, opts *CreateOperationOptions) (*Operation, error)
}

type CreateOperationOptions struct {
	ID        string
	Operation *Operation
}

type OperationDeleter interface {
	DeleteOperation(ctx context.Context, opts *DeleteOperationOptions) error
}

type DeleteOperationOptions struct {
	Name string
}

type OperationLister interface {
	ListOperations(ctx context.Context, opts *ListOperationsOptions) (*ListOperationsResult, error)
}

type ListOperationsOptions struct {
	Filter               string
	PageSize             int32
	PageToken            string
	ReturnPartialSuccess bool
}

type ListOperationsResult struct {
	Operations    []*Operation
	NextPageToken string
	Unreachable   []string
}

type OperationUpdater interface {
	UpdateOperation(ctx context.Context, opts *UpdateOperationOptions) (*Operation, error)
}

type UpdateOperationOptions struct {
	Operation *Operation
	Paths     []string
}
type OperationWaiter interface {
	WaitOperation(ctx context.Context, opts *WaitOperationOptions) (*Operation, error)
}

type WaitOperationOptions struct {
	Name    string
	Timeout time.Duration
}

type OperationGetter interface {
	GetOperation(ctx context.Context, opts *GetOperationOptions) (*Operation, error)
}

type GetOperationOptions struct {
	Name string
}
