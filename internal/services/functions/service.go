package funcsrv

import (
	"context"
	"fmt"

	funcdomain "github.com/10Narratives/faas/internal/domain/functions"
)

type ObjectRepository interface {
	funcdomain.FunctionSourceUploader
}

type MetadataRepository interface {
	funcdomain.FunctionGetter
}

type Service struct {
	objRepo  ObjectRepository
	metaRepo MetadataRepository
}

func NewService() (*Service, error) {
	return nil, nil
}

func (s *Service) GetFunction(ctx context.Context, args *funcdomain.GetFunctionArgs) (*funcdomain.GetFunctionResult, error) {
	return nil, nil
}

func (s *Service) UploadFunctionSource(ctx context.Context, args *funcdomain.UploadFunctionSourceArgs) (*funcdomain.UploadFunctionSourceResult, error) {
	fmt.Println(args.Metadata.FunctionName)

	return &funcdomain.UploadFunctionSourceResult{}, nil
}
