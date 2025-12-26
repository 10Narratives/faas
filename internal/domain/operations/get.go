package opdomain

import "context"

type OperationGetter interface {
	GetOperation(ctx context.Context, opts *GetOperationOptions) (*Operation, error)
}

type GetOperationOptions struct {
	Name string
}
