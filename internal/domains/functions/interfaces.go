package funcdomain

import (
	"context"
	"encoding/json"
	"io"
)

type FunctionUploader interface {
	UploadFunction(ctx context.Context, args *UploadFunctionArgs) (*UploadFunctionResult, error)
}

type FunctionGetter interface {
	GetFunction(ctx context.Context, args *GetFunctionArgs) (*GetFunctionResult, error)
}

type FunctionLister interface {
	ListFunctions(ctx context.Context, args *ListFunctionsArgs) (*ListFunctionsResult, error)
}

type FunctionDeleter interface {
	DeleteFunction(ctx context.Context, args *DeleteFunctionArgs) error
}

type FunctionExecutor interface {
	ExecuteFunction(ctx context.Context, args *ExecuteFunctionArgs) (*ExecuteFunctionResult, error)
}

type UploadFunctionArgs struct {
	Name        FunctionName
	DisplayName string
	Format      UploadFunctionFormat
	Data        io.ReadCloser
}

type UploadFunctionResult struct {
	Function *Function
}

type GetFunctionArgs struct {
	Name FunctionName
}

type GetFunctionResult struct {
	Function *Function
}

type ListFunctionsArgs struct {
	PageSize  int32
	PageToken string
}

type ListFunctionsResult struct {
	Functions     []*Function
	NextPageToken string
}

type DeleteFunctionArgs struct {
	Name FunctionName
}

type ExecuteFunctionArgs struct {
	Name       FunctionName
	Parameters json.RawMessage
}

type ExecuteFunctionResult struct {
	OperationName string
}
