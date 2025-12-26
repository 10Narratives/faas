package opdomain

import "context"

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
