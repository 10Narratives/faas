package funcdomain

import (
	"bytes"
	"context"
)

type FunctionGetter interface {
	GetFunction(ctx context.Context, opts *GetFunctionOptions) (*Function, error)
}

type GetFunctionOptions struct {
	Name string
}

type FunctionUploader interface {
	UploadFunction(ctx context.Context, opts *UploadFunctionOptions) (*Function, error)
}

type UploadFunctionOptions struct {
	Function     *Function
	FunctionData *bytes.Buffer
}
