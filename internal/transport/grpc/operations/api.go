package opapi

import (
	"context"
	"errors"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	opdomain "github.com/10Narratives/faas/internal/domain/operations"
	grpctr "github.com/10Narratives/faas/internal/transport/grpc"
	sliceutils "github.com/10Narratives/faas/pkg/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type OperationService interface {
	opdomain.OperationCanceler
	opdomain.OperationDeleter
	opdomain.OperationGetter
	opdomain.OperationLister
	opdomain.OperationWaiter
}

type api struct {
	longrunningpb.UnimplementedOperationsServer
	operationService OperationService
}

func NewRegistration(operationService OperationService) grpctr.ServiceRegistration {
	return func(s *grpc.Server) {
		longrunningpb.RegisterOperationsServer(s, &api{operationService: operationService})
	}
}

func (a *api) CancelOperation(ctx context.Context, req *longrunningpb.CancelOperationRequest) (*emptypb.Empty, error) {
	options := &opdomain.CancelOperationOptions{Name: req.GetName()}
	if err := a.operationService.CancelOperation(ctx, options); err != nil {
		return nil, errorToProto(err)
	}
	return &emptypb.Empty{}, nil
}

func (a *api) DeleteOperation(ctx context.Context, req *longrunningpb.DeleteOperationRequest) (*emptypb.Empty, error) {
	options := &opdomain.DeleteOperationOptions{Name: req.GetName()}
	if err := a.operationService.DeleteOperation(ctx, options); err != nil {
		return nil, errorToProto(err)
	}
	return &emptypb.Empty{}, nil
}

func (a *api) GetOperation(ctx context.Context, req *longrunningpb.GetOperationRequest) (*longrunningpb.Operation, error) {
	options := &opdomain.GetOperationOptions{Name: req.GetName()}
	operation, err := a.operationService.GetOperation(ctx, options)
	if err != nil {
		return nil, errorToProto(err)
	}

	return operationToProto(operation), nil
}

func (a *api) ListOperations(ctx context.Context, req *longrunningpb.ListOperationsRequest) (*longrunningpb.ListOperationsResponse, error) {
	options := &opdomain.ListOperationsOptions{
		Filter:               req.GetFilter(),
		PageSize:             req.GetPageSize(),
		PageToken:            req.GetPageToken(),
		ReturnPartialSuccess: req.GetReturnPartialSuccess(),
	}

	listResult, err := a.operationService.ListOperations(ctx, options)
	if err != nil {
		return nil, errorToProto(err)
	}

	converted := sliceutils.Map(listResult.Operations, operationToProto)

	return &longrunningpb.ListOperationsResponse{
		Operations:    converted,
		NextPageToken: listResult.NextPageToken,
		Unreachable:   listResult.Unreachable,
	}, nil
}

func (a *api) WaitOperation(ctx context.Context, req *longrunningpb.WaitOperationRequest) (*longrunningpb.Operation, error) {
	options := &opdomain.WaitOperationOptions{
		Name:    req.GetName(),
		Timeout: req.GetTimeout().AsDuration(),
	}

	operation, err := a.operationService.WaitOperation(ctx, options)
	if err != nil {
		return nil, errorToProto(err)
	}

	return operationToProto(operation), nil
}

func errorToProto(err error) error {
	switch {
	case errors.Is(err, opdomain.ErrOperationNotFound):
		return status.Error(codes.NotFound, "operation not found")
	default:
		return status.Error(codes.Internal, "cannot cancel operation")
	}
}

func operationToProto(operation *opdomain.Operation) *longrunningpb.Operation {
	if operation == nil {
		return nil
	}

	return &longrunningpb.Operation{
		Name: operation.Name,
	}
}
