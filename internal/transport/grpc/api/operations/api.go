package opapi

import (
	"context"
	"errors"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	opdomain "github.com/10Narratives/faas/internal/domains/operations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockery --name OperationService --output ./mocks --outpkg mocks --with-expecter --filename operation_service.go
type OperationService interface {
	opdomain.OperationGetter
	opdomain.OperationLister
	opdomain.OperationWaiter
	opdomain.OperationCanceler
	opdomain.OperationDeleter
}

type Server struct {
	longrunningpb.UnimplementedOperationsServer
	operationService OperationService
}

func NewServer(operationService OperationService) *Server {
	return &Server{operationService: operationService}
}

func NewRegistration(operationService OperationService) grpcsrv.ServiceRegistration {
	return func(s *grpc.Server) {
		longrunningpb.RegisterOperationsServer(s, NewServer(operationService))
	}
}

func (s *Server) ListOperations(ctx context.Context, req *longrunningpb.ListOperationsRequest) (*longrunningpb.ListOperationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	res, err := s.operationService.ListOperations(ctx, &opdomain.ListOperationsArgs{
		Name:                 req.GetName(),
		Filter:               req.GetFilter(),
		PageSize:             req.GetPageSize(),
		PageToken:            req.GetPageToken(),
		ReturnPartialSuccess: req.GetReturnPartialSuccess(),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	if res == nil {
		return nil, status.Error(codes.Internal, "missing result")
	}

	return &longrunningpb.ListOperationsResponse{
		Operations:    res.Operations,
		NextPageToken: res.NextPageToken,
		Unreachable:   res.Unreachable,
	}, nil
}

func (s *Server) GetOperation(ctx context.Context, req *longrunningpb.GetOperationRequest) (*longrunningpb.Operation, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	name, err := opdomain.ParseOperationName(req.GetName())
	if err != nil {
		return nil, toStatusErr(err)
	}

	res, err := s.operationService.GetOperation(ctx, &opdomain.GetOperationArgs{Name: name})
	if err != nil {
		return nil, toStatusErr(err)
	}
	if res == nil || res.Operation == nil {
		return nil, status.Error(codes.Internal, "missing operation in result")
	}

	return res.Operation, nil
}

func (s *Server) DeleteOperation(ctx context.Context, req *longrunningpb.DeleteOperationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	name, err := opdomain.ParseOperationName(req.GetName())
	if err != nil {
		return nil, toStatusErr(err)
	}

	if err := s.operationService.DeleteOperation(ctx, &opdomain.DeleteOperationArgs{Name: name}); err != nil {
		return nil, toStatusErr(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) CancelOperation(ctx context.Context, req *longrunningpb.CancelOperationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	name, err := opdomain.ParseOperationName(req.GetName())
	if err != nil {
		return nil, toStatusErr(err)
	}

	if err := s.operationService.CancelOperation(ctx, &opdomain.CancelOperationArgs{Name: name}); err != nil {
		return nil, toStatusErr(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) WaitOperation(ctx context.Context, req *longrunningpb.WaitOperationRequest) (*longrunningpb.Operation, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	name, err := opdomain.ParseOperationName(req.GetName())
	if err != nil {
		return nil, toStatusErr(err)
	}

	args := &opdomain.WaitOperationArgs{Name: name}
	if req.GetTimeout() != nil {
		args.Timeout = req.GetTimeout().AsDuration()
	}

	res, err := s.operationService.WaitOperation(ctx, args)
	if err != nil {
		return nil, toStatusErr(err)
	}
	if res == nil || res.Operation == nil {
		return nil, status.Error(codes.Internal, "missing operation in result")
	}

	return res.Operation, nil
}

func toStatusErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}

	switch {
	case errors.Is(err, opdomain.ErrOperationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, opdomain.ErrInvalidOperationName),
		errors.Is(err, opdomain.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
