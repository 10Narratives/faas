package opsrv

import (
	"context"
	"errors"
	"fmt"

	opdomain "github.com/10Narratives/faas/internal/domain/operations"
	"github.com/google/uuid"
)

type OperationRepository interface {
	opdomain.OperationGetter
	opdomain.OperationLister
	opdomain.OperationDeleter
	opdomain.OperationCreator
	opdomain.OperationUpdater
}

type Service struct {
	operationRepository OperationRepository
}

func NewService(
	operationRepository OperationRepository,
) (*Service, error) {
	if operationRepository == nil {
		return nil, errors.New("operation repository is required")
	}

	return &Service{
		operationRepository: operationRepository,
	}, nil
}

func (s *Service) CreateOperation(ctx context.Context, opts *opdomain.CreateOperationOptions) (*opdomain.Operation, error) {
	if opts == nil {
		return nil, errors.New("create operation options are required")
	}

	if opts.ID == "" {
		uid, err := uuid.NewV7()
		if err != nil {
			return nil, errors.New("cannot generate id for new operation")
		}
		opts.ID = uid.String()
	}

	return s.operationRepository.CreateOperation(ctx, opts)
}

func (s *Service) GetOperation(ctx context.Context, opts *opdomain.GetOperationOptions) (*opdomain.Operation, error) {
	if opts == nil {
		return nil, fmt.Errorf("%w: options are required", opdomain.ErrFailedPrecondition)
	}

	if opts.Name == "" {
		return nil, fmt.Errorf("%w: operation name is required", opdomain.ErrFailedPrecondition)
	}

	return s.operationRepository.GetOperation(ctx, opts)
}

func (s *Service) ListOperations(ctx context.Context, opts *opdomain.ListOperationsOptions) (*opdomain.ListOperationsResult, error) {
	if opts == nil {
		opts = &opdomain.ListOperationsOptions{}
	}

	if opts.PageSize < 1 {
		opts.PageSize = 50
	}

	opts.PageSize = min(opts.PageSize, 1000)

	return s.operationRepository.ListOperations(ctx, opts)
}

func (s *Service) DeleteOperation(ctx context.Context, opts *opdomain.DeleteOperationOptions) error {
	if opts == nil {
		return fmt.Errorf("%w: options are required", opdomain.ErrFailedPrecondition)
	}

	if opts.Name == "" {
		return fmt.Errorf("%w: operation name is required", opdomain.ErrFailedPrecondition)
	}

	return s.operationRepository.DeleteOperation(ctx, opts)
}

func (s *Service) UpdateOperation(ctx context.Context, opts *opdomain.UpdateOperationOptions) (*opdomain.Operation, error) {
	if opts == nil {
		return nil, fmt.Errorf("%w: options are required", opdomain.ErrFailedPrecondition)
	}

	return s.operationRepository.UpdateOperation(ctx, opts)
}
