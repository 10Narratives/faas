package opdomain

import (
	"context"
	"time"

	longrunning "cloud.google.com/go/longrunning/autogen/longrunningpb"
)

type OperationGetter interface {
	GetOperation(ctx context.Context, args *GetOperationArgs) (*GetOperationResult, error)
}

type GetOperationArgs struct {
	Name OperationName
}

type GetOperationResult struct {
	Operation *longrunning.Operation
}

type OperationLister interface {
	ListOperations(ctx context.Context, args *ListOperationsArgs) (*ListOperationsResult, error)
}

type ListOperationsArgs struct {
	Name                 string
	Filter               string
	PageSize             int32
	PageToken            string
	ReturnPartialSuccess bool
}

type ListOperationsResult struct {
	Operations    []*longrunning.Operation
	NextPageToken string
	Unreachable   []string
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
	Operation *longrunning.Operation
}
