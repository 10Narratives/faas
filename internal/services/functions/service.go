package funcsrv

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	funcdomain "github.com/10Narratives/faas/internal/domains/functions"
	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	"github.com/google/uuid"
)

type FunctionMetadataRepository interface {
	CreateFunction(ctx context.Context, fn *funcdomain.Function) error
	funcdomain.FunctionGetter
	funcdomain.FunctionDeleter
	funcdomain.FunctionLister
}

type FunctionObjectRepository interface {
	SaveBundle(ctx context.Context, name funcdomain.FunctionName, format funcdomain.UploadFunctionFormat, data io.ReadCloser) (*funcdomain.SourceBundle, error)
	OpenBundle(ctx context.Context, bundle *funcdomain.SourceBundle) (io.ReadCloser, error)
	DeleteBundle(ctx context.Context, bundle *funcdomain.SourceBundle) error
}

type TaskService interface {
	taskdomain.TaskCreator
}

type Service struct {
	funcMetaRepo FunctionMetadataRepository
	funcObjRepo  FunctionObjectRepository
	taskService  TaskService
}

func NewService(
	funcMetaRepo FunctionMetadataRepository,
	funcObjRepo FunctionObjectRepository,
	taskService TaskService,
) *Service {
	return &Service{
		funcMetaRepo: funcMetaRepo,
		funcObjRepo:  funcObjRepo,
		taskService:  taskService,
	}
}

func (s *Service) DeleteFunction(ctx context.Context, args *funcdomain.DeleteFunctionArgs) error {
	if args == nil {
		return funcdomain.ErrInvalidArgument
	}

	got, err := s.funcMetaRepo.GetFunction(ctx, &funcdomain.GetFunctionArgs{Name: args.Name})
	if err != nil {
		return err
	}

	if err := s.funcMetaRepo.DeleteFunction(ctx, args); err != nil {
		return err
	}

	if err := s.funcObjRepo.DeleteBundle(ctx, got.Function.Bundle); err != nil {
		return err
	}

	return nil
}

func (s *Service) ExecuteFunction(ctx context.Context, args *funcdomain.ExecuteFunctionArgs) (*funcdomain.ExecuteFunctionResult, error) {
	if args == nil {
		return nil, funcdomain.ErrInvalidArgument
	}
	if args.Name == "" {
		return nil, funcdomain.ErrInvalidArgument
	}

	if len(args.Parameters) != 0 {
		var tmp any
		if err := json.Unmarshal([]byte(args.Parameters), &tmp); err != nil {
			return nil, funcdomain.ErrInvalidArgument
		}
	}

	got, err := s.funcMetaRepo.GetFunction(ctx, &funcdomain.GetFunctionArgs{Name: args.Name})
	if err != nil {
		return nil, err
	}
	if got == nil || got.Function == nil {
		return nil, funcdomain.ErrFunctionNotFound
	}

	res, err := s.taskService.CreateTask(ctx, &taskdomain.CreateTaskArgs{Function: string(args.Name), Parameters: string(args.Parameters)})
	if err != nil {
		return nil, err
	}

	return &funcdomain.ExecuteFunctionResult{
		TaskName: res.Name,
	}, nil
}

func (s *Service) GetFunction(ctx context.Context, args *funcdomain.GetFunctionArgs) (*funcdomain.GetFunctionResult, error) {
	if args == nil {
		return nil, funcdomain.ErrInvalidArgument
	}

	res, err := s.funcMetaRepo.GetFunction(ctx, args)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Function == nil {
		return nil, funcdomain.ErrFunctionNotFound
	}

	return res, nil
}

func (s *Service) ListFunctions(ctx context.Context, args *funcdomain.ListFunctionsArgs) (*funcdomain.ListFunctionsResult, error) {
	if args == nil {
		return nil, funcdomain.ErrInvalidArgument
	}

	if args.PageSize <= 0 {
		args.PageSize = 50
	}

	if args.PageSize > 1000 {
		args.PageSize = 1000
	}

	return s.funcMetaRepo.ListFunctions(ctx, args)
}

func (s *Service) UploadFunction(ctx context.Context, args *funcdomain.UploadFunctionArgs) (*funcdomain.UploadFunctionResult, error) {
	if args == nil {
		return nil, funcdomain.ErrInvalidArgument
	}
	if args.Name == "" {
		return nil, funcdomain.ErrInvalidArgument
	}
	if args.Data == nil {
		return nil, funcdomain.ErrInvalidArgument
	}
	if !isSupportedFormat(args.Format) {
		return nil, funcdomain.ErrUnsupportedFormat
	}

	bundle, err := s.funcObjRepo.SaveBundle(ctx, args.Name, args.Format, args.Data)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, funcdomain.ErrInvalidArgument
	}

	fn := &funcdomain.Function{
		InternalID:  uuid.New(),
		Name:        args.Name,
		DisplayName: args.DisplayName,
		UploadedAt:  time.Now().UTC(),
		Bundle:      bundle,
	}

	if err := s.funcMetaRepo.CreateFunction(ctx, fn); err != nil {
		if errors.Is(err, funcdomain.ErrFunctionAlreadyExists) {
			_ = s.funcObjRepo.DeleteBundle(ctx, bundle)
		}
		return nil, err
	}

	return &funcdomain.UploadFunctionResult{Function: fn}, nil
}

func isSupportedFormat(f funcdomain.UploadFunctionFormat) bool {
	switch f {
	case funcdomain.ZipFormat, funcdomain.TarGZFormat:
		return true
	default:
		return false
	}
}
