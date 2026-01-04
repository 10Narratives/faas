package funcdomain

import (
	"bytes"
	"context"
)

type FunctionSourceUploader interface {
	UploadFunctionSource(ctx context.Context, args *UploadFunctionSourceArgs) (*UploadFunctionSourceResult, error)
}

type UploadFunctionSourceArgs struct {
	Metadata UploadFunctionSourceMetadata
	Data     *bytes.Buffer
}

type UploadFunctionSourceMetadata struct {
	FunctionName         string
	SourceBundleMetadata SourceBundleMetadata
}

type SourceBundleType int

const (
	SourceBundleTypeUnspecified SourceBundleType = iota
	SourceBundleTypeZip
)

type SourceBundleMetadata struct {
	Type     SourceBundleType
	FileName string
	Size     uint64
}

type UploadFunctionSourceResult struct {
	FunctionID string
	ObjectKey  string
}

type FunctionGetter interface {
	GetFunction(ctx context.Context, args *GetFunctionArgs) (*GetFunctionResult, error)
}

type GetFunctionArgs struct {
	Name string
}

type GetFunctionResult struct {
	Function *Function
}
