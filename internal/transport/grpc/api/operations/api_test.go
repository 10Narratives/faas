package opapi_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	opdomain "github.com/10Narratives/faas/internal/domains/operations"
	opapi "github.com/10Narratives/faas/internal/transport/grpc/api/operations"
	opapimocks "github.com/10Narratives/faas/internal/transport/grpc/api/operations/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServer_ListOperations_NilReq(t *testing.T) {
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	_, err := srv.ListOperations(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_ListOperations_ServiceErrorMapped(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	req := &longrunningpb.ListOperationsRequest{
		Name:                 "parent",
		Filter:               "f",
		PageSize:             10,
		PageToken:            "pt",
		ReturnPartialSuccess: true,
	}

	svc.EXPECT().
		ListOperations(ctx, mock.MatchedBy(func(a *opdomain.ListOperationsArgs) bool {
			return a != nil &&
				a.Name == req.GetName() &&
				a.Filter == req.GetFilter() &&
				a.PageSize == req.GetPageSize() &&
				a.PageToken == req.GetPageToken() &&
				a.ReturnPartialSuccess == req.GetReturnPartialSuccess()
		})).
		Return(nil, opdomain.ErrInvalidArgument)

	_, err := srv.ListOperations(ctx, req)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_ListOperations_MissingResult(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	req := &longrunningpb.ListOperationsRequest{Name: "parent"}

	svc.EXPECT().
		ListOperations(ctx, mock.Anything).
		Return(nil, nil)

	_, err := srv.ListOperations(ctx, req)
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "missing result")
}

func TestServer_ListOperations_OK(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	op1 := &longrunningpb.Operation{Name: "operations/1", Done: false}
	op2 := &longrunningpb.Operation{Name: "operations/2", Done: true}

	req := &longrunningpb.ListOperationsRequest{
		Name:                 "parent",
		Filter:               "state=done",
		PageSize:             2,
		PageToken:            "t",
		ReturnPartialSuccess: true,
	}

	svc.EXPECT().
		ListOperations(ctx, mock.MatchedBy(func(a *opdomain.ListOperationsArgs) bool {
			return a != nil &&
				a.Name == req.GetName() &&
				a.Filter == req.GetFilter() &&
				a.PageSize == req.GetPageSize() &&
				a.PageToken == req.GetPageToken() &&
				a.ReturnPartialSuccess == req.GetReturnPartialSuccess()
		})).
		Return(&opdomain.ListOperationsResult{
			Operations:    []*longrunningpb.Operation{op1, op2},
			NextPageToken: "next",
			Unreachable:   []string{"u1"},
		}, nil)

	res, err := srv.ListOperations(ctx, req)
	require.NoError(t, err)
	require.Len(t, res.GetOperations(), 2)
	require.Equal(t, "next", res.GetNextPageToken())
	require.Equal(t, []string{"u1"}, res.GetUnreachable())
	require.Equal(t, "operations/1", res.GetOperations()[0].GetName())
}

func TestServer_GetOperation_NilReq(t *testing.T) {
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	_, err := srv.GetOperation(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_GetOperation_InvalidName(t *testing.T) {
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	_, err := srv.GetOperation(context.Background(), &longrunningpb.GetOperationRequest{Name: "bad"})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_GetOperation_NotFoundMapped(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	name := "operations/123"
	req := &longrunningpb.GetOperationRequest{Name: name}

	svc.EXPECT().
		GetOperation(ctx, mock.MatchedBy(func(a *opdomain.GetOperationArgs) bool {
			return a != nil && a.Name == opdomain.OperationName(name)
		})).
		Return(nil, opdomain.ErrOperationNotFound)

	_, err := srv.GetOperation(ctx, req)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestServer_GetOperation_MissingOperationInResult(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	name := "operations/123"
	req := &longrunningpb.GetOperationRequest{Name: name}

	svc.EXPECT().
		GetOperation(ctx, mock.Anything).
		Return(&opdomain.GetOperationResult{Operation: nil}, nil)

	_, err := srv.GetOperation(ctx, req)
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "missing operation")
}

func TestServer_GetOperation_OK(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	name := "operations/123"
	req := &longrunningpb.GetOperationRequest{Name: name}

	op := &longrunningpb.Operation{Name: name, Done: false}

	svc.EXPECT().
		GetOperation(ctx, mock.MatchedBy(func(a *opdomain.GetOperationArgs) bool {
			return a != nil && a.Name == opdomain.OperationName(name)
		})).
		Return(&opdomain.GetOperationResult{Operation: op}, nil)

	got, err := srv.GetOperation(ctx, req)
	require.NoError(t, err)
	require.Equal(t, op, got)
}

func TestServer_DeleteOperation_NilReq(t *testing.T) {
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	_, err := srv.DeleteOperation(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_DeleteOperation_InvalidName(t *testing.T) {
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	_, err := srv.DeleteOperation(context.Background(), &longrunningpb.DeleteOperationRequest{Name: "bad"})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_DeleteOperation_OK(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	name := "operations/123"
	req := &longrunningpb.DeleteOperationRequest{Name: name}

	svc.EXPECT().
		DeleteOperation(ctx, mock.MatchedBy(func(a *opdomain.DeleteOperationArgs) bool {
			return a != nil && a.Name == opdomain.OperationName(name)
		})).
		Return(nil)

	res, err := srv.DeleteOperation(ctx, req)
	require.NoError(t, err)
	require.Equal(t, &emptypb.Empty{}, res)
}

func TestServer_CancelOperation_NilReq(t *testing.T) {
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	_, err := srv.CancelOperation(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_CancelOperation_ContextCanceledMapped(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	name := "operations/123"
	req := &longrunningpb.CancelOperationRequest{Name: name}

	svc.EXPECT().
		CancelOperation(ctx, mock.Anything).
		Return(context.Canceled)

	_, err := srv.CancelOperation(ctx, req)
	require.Error(t, err)
	require.Equal(t, codes.Canceled, status.Code(err))
}

func TestServer_CancelOperation_OK(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	name := "operations/123"
	req := &longrunningpb.CancelOperationRequest{Name: name}

	svc.EXPECT().
		CancelOperation(ctx, mock.MatchedBy(func(a *opdomain.CancelOperationArgs) bool {
			return a != nil && a.Name == opdomain.OperationName(name)
		})).
		Return(nil)

	res, err := srv.CancelOperation(ctx, req)
	require.NoError(t, err)
	require.Equal(t, &emptypb.Empty{}, res)
}

func TestServer_WaitOperation_NilReq(t *testing.T) {
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	_, err := srv.WaitOperation(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_WaitOperation_InvalidName(t *testing.T) {
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	_, err := srv.WaitOperation(context.Background(), &longrunningpb.WaitOperationRequest{Name: "bad"})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_WaitOperation_TimeoutPassedToDomain(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	name := "operations/123"
	req := &longrunningpb.WaitOperationRequest{
		Name:    name,
		Timeout: durationpb.New(2 * time.Second),
	}

	op := &longrunningpb.Operation{Name: name, Done: true}

	svc.EXPECT().
		WaitOperation(ctx, mock.MatchedBy(func(a *opdomain.WaitOperationArgs) bool {
			return a != nil &&
				a.Name == opdomain.OperationName(name) &&
				a.Timeout == 2*time.Second
		})).
		Return(&opdomain.WaitOperationResult{Operation: op}, nil)

	got, err := srv.WaitOperation(ctx, req)
	require.NoError(t, err)
	require.Equal(t, op, got)
}

func TestServer_WaitOperation_UnknownErrorMappedToInternal(t *testing.T) {
	ctx := context.Background()
	svc := opapimocks.NewOperationService(t)
	srv := opapi.NewServer(svc)

	name := "operations/123"
	req := &longrunningpb.WaitOperationRequest{Name: name}

	svc.EXPECT().
		WaitOperation(ctx, mock.Anything).
		Return(nil, errors.New("boom"))

	_, err := srv.WaitOperation(ctx, req)
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}
