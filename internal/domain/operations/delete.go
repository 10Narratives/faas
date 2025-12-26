package opdomain

import "context"

type OperationDeleter interface {
	DeleteOperation(ctx context.Context, opts *DeleteOperationOptions) error
}

type DeleteOperationOptions struct {
	Name string
}
