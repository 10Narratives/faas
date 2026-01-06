package opdomain

import (
	"context"
	"time"
)

type OperationGetter interface {
	GetOperation(ctx context.Context, args *GetOperationArgs) (*GetOperationResult, error)
}

type GetOperationArgs struct {
	Name OperationName
}

type GetOperationResult struct {
	Operation *Operation
}

type OperationCanceler interface {
	CancelOperation(ctx context.Context, args *CancelOperationArgs) error
}

type CancelOperationArgs struct {
	Name OperationName
}

type OperationDeleter interface {
	DeleteOperation(ctx context.Context, args *DeleteOperationArgs) error
}

type DeleteOperationArgs struct {
	Name OperationName
}

type OperationWaiter interface {
	WaitOperation(ctx context.Context, args *WaitOperationArgs) (*WaitOperationResult, error)
}

type WaitOperationArgs struct {
	Name    OperationName
	Timeout time.Duration
}

type WaitOperationResult struct {
	Operation *Operation
}
