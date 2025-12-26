package opdomain

import "context"

type OperationUpdater interface {
	UpdateOperation(ctx context.Context, opts *UpdateOperationOptions) (*Operation, error)
}

type UpdateOperationOptions struct {
	Operation *Operation
	Paths     []string
}
