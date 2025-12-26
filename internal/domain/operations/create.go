package opdomain

import (
	"context"
)

type OperationCreator interface {
	CreateOperation(ctx context.Context, opts *CreateOperationOptions) (*Operation, error)
}

type CreateOperationOptions struct {
	ID        string
	Operation *Operation
}
