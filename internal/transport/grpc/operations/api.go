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

//mockery:generate: true
type OperationService interface {
	opdomain.OperationCanceler
	opdomain.OperationDeleter
	opdomain.OperationGetter
	opdomain.OperationLister
	opdomain.OperationWaiter
}

type Server struct {
	longrunningpb.UnimplementedOperationsServer
	operationService OperationService
}

func NewServer(operationService OperationService) *Server {
	return &Server{operationService: operationService}
}

func NewRegistration(operationService OperationService) grpctr.ServiceRegistration {
	return func(s *grpc.Server) {
		longrunningpb.RegisterOperationsServer(s, NewServer(operationService))
	}
}

func (s *Server) CancelOperation(ctx context.Context, req *longrunningpb.CancelOperationRequest) (*emptypb.Empty, error) {
	opts := &opdomain.CancelOperationOptions{Name: req.GetName()}
	if opts.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "operation name is required")
	}

	if err := s.operationService.CancelOperation(ctx, opts); err != nil {
		return nil, errorToProto(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) DeleteOperation(ctx context.Context, req *longrunningpb.DeleteOperationRequest) (*emptypb.Empty, error) {
	opts := &opdomain.DeleteOperationOptions{Name: req.GetName()}
	if opts.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "operation name is required")
	}

	if err := s.operationService.DeleteOperation(ctx, opts); err != nil {
		return nil, errorToProto(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetOperation(ctx context.Context, req *longrunningpb.GetOperationRequest) (*longrunningpb.Operation, error) {
	opts := &opdomain.GetOperationOptions{Name: req.GetName()}
	if opts.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "operation name is required")
	}

	operation, err := s.operationService.GetOperation(ctx, opts)
	if err != nil {
		return nil, errorToProto(err)
	}

	return operationToProto(operation), nil
}

func (s *Server) ListOperations(ctx context.Context, req *longrunningpb.ListOperationsRequest) (*longrunningpb.ListOperationsResponse, error) {
	options := &opdomain.ListOperationsOptions{
		Filter:               req.GetFilter(),
		PageSize:             req.GetPageSize(),
		PageToken:            req.GetPageToken(),
		ReturnPartialSuccess: req.GetReturnPartialSuccess(),
	}

	listResult, err := s.operationService.ListOperations(ctx, options)
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

func (s *Server) WaitOperation(ctx context.Context, req *longrunningpb.WaitOperationRequest) (*longrunningpb.Operation, error) {
	options := &opdomain.WaitOperationOptions{
		Name:    req.GetName(),
		Timeout: req.GetTimeout().AsDuration(),
	}

	operation, err := s.operationService.WaitOperation(ctx, options)
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
