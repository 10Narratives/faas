package opdomain

import "context"

type OperationCanceler interface {
	CancelOperation(ctx context.Context, opts *CancelOperationOptions) error
}

type CancelOperationOptions struct {
	Name string
}
