package funcapi

import (
	"context"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	funcdomain "github.com/10Narratives/faas/internal/domain/functions"
	opdomain "github.com/10Narratives/faas/internal/domain/operations"
	grpctr "github.com/10Narratives/faas/internal/transport/grpc"
	"github.com/10Narratives/faas/pkg/faas/functions/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FunctionService interface {
	ImportFunctions(ctx context.Context, sources *funcdomain.ArchiveSource) (*opdomain.Operation, error)
}

type api struct {
	functions.UnimplementedFunctionServiceServer
	functionService FunctionService
}

func NewRegistration(functionService FunctionService) grpctr.ServiceRegistration {
	return func(s *grpc.Server) {
		functions.RegisterFunctionServiceServer(s, &api{functionService: functionService})
	}
}

func (a *api) ImportFunctions(ctx context.Context, req *functions.ImportFunctionsRequest) (*longrunningpb.Operation, error) {
	return nil, status.Error(codes.Unimplemented, "rpc method is not implemented")
}
