package opdomain

import (
	"context"
	"time"
)

type OperationWaiter interface {
	WaitOperation(ctx context.Context, opts *WaitOperationOptions) (*Operation, error)
}

type WaitOperationOptions struct {
	Name    string
	Timeout time.Duration
}
