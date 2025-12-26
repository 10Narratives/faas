package opapi_test

import (
	"context"
	"testing"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	opdomain "github.com/10Narratives/faas/internal/domain/operations"
	opapi "github.com/10Narratives/faas/internal/transport/grpc/operations"
	"github.com/10Narratives/faas/internal/transport/grpc/operations/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServer_CancelOperation(t *testing.T) {
	t.Parallel()

	type fields struct {
		setupOperationService func(m *mocks.OperationService)
	}

	type args struct {
		ctx context.Context
		req *longrunningpb.CancelOperationRequest
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    assert.ValueAssertionFunc
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "invalid request",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.CancelOperationRequest{Name: ""},
			},
			want: assert.Nil,
			wantErr: func(tt assert.TestingT, err error, i ...interface{}) bool {
				stat, ok := status.FromError(err)
				return assert.True(t, ok) && assert.Equal(t, codes.InvalidArgument, stat.Code())
			},
		},
		{
			name: "operation not found",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {
					m.On("CancelOperation", mock.Anything, mock.Anything).Return(opdomain.ErrOperationNotFound)
				},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.CancelOperationRequest{Name: "operations/123"},
			},
			want: assert.Nil,
			wantErr: func(tt assert.TestingT, err error, i ...interface{}) bool {
				stat, ok := status.FromError(err)
				return assert.True(t, ok) && assert.Equal(t, codes.NotFound, stat.Code())
			},
		},
		{
			name: "operation canceled",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {
					m.On("CancelOperation", mock.Anything, mock.Anything).Return(nil)
				},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.CancelOperationRequest{Name: "operations/123"},
			},
			want:    assert.NotNil,
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockOperationService := mocks.NewOperationService(t)
			tt.fields.setupOperationService(mockOperationService)

			server := opapi.NewServer(mockOperationService)
			resp, err := server.CancelOperation(tt.args.ctx, tt.args.req)

			tt.want(t, resp)
			tt.wantErr(t, err)

			mockOperationService.AssertExpectations(t)
		})
	}
}

func TestServer_DeleteOperation(t *testing.T) {
	t.Parallel()

	type fields struct {
		setupOperationService func(m *mocks.OperationService)
	}

	type args struct {
		ctx context.Context
		req *longrunningpb.DeleteOperationRequest
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    assert.ValueAssertionFunc
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "invalid request",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.DeleteOperationRequest{Name: ""},
			},
			want: assert.Nil,
			wantErr: func(tt assert.TestingT, err error, i ...interface{}) bool {
				stat, ok := status.FromError(err)
				return assert.True(t, ok) && assert.Equal(t, codes.InvalidArgument, stat.Code())
			},
		},
		{
			name: "operation not found",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {
					m.On("DeleteOperation", mock.Anything, mock.Anything).Return(opdomain.ErrOperationNotFound)
				},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.DeleteOperationRequest{Name: "operations/123"},
			},
			want: assert.Nil,
			wantErr: func(tt assert.TestingT, err error, i ...interface{}) bool {
				stat, ok := status.FromError(err)
				return assert.True(t, ok) && assert.Equal(t, codes.NotFound, stat.Code())
			},
		},
		{
			name: "operation canceled",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {
					m.On("DeleteOperation", mock.Anything, mock.Anything).Return(nil)
				},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.DeleteOperationRequest{Name: "operations/123"},
			},
			want:    assert.NotNil,
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockOperationService := mocks.NewOperationService(t)
			tt.fields.setupOperationService(mockOperationService)

			server := opapi.NewServer(mockOperationService)
			resp, err := server.DeleteOperation(tt.args.ctx, tt.args.req)

			tt.want(t, resp)
			tt.wantErr(t, err)

			mockOperationService.AssertExpectations(t)
		})
	}
}

func TestServer_GetOperation(t *testing.T) {
	t.Parallel()

	type fields struct {
		setupOperationService func(m *mocks.OperationService)
	}

	type args struct {
		ctx context.Context
		req *longrunningpb.GetOperationRequest
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    assert.ValueAssertionFunc
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "invalid request",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.GetOperationRequest{Name: ""},
			},
			want: assert.Nil,
			wantErr: func(tt assert.TestingT, err error, i ...interface{}) bool {
				stat, ok := status.FromError(err)
				return assert.True(t, ok) && assert.Equal(t, codes.InvalidArgument, stat.Code())
			},
		},
		{
			name: "operation not found",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {
					m.On("GetOperation", mock.Anything, mock.Anything).Return(nil, opdomain.ErrOperationNotFound)
				},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.GetOperationRequest{Name: "operations/123"},
			},
			want: assert.Nil,
			wantErr: func(tt assert.TestingT, err error, i ...interface{}) bool {
				stat, ok := status.FromError(err)
				return assert.True(t, ok) && assert.Equal(t, codes.NotFound, stat.Code())
			},
		},
		{
			name: "operation canceled",
			fields: fields{
				setupOperationService: func(m *mocks.OperationService) {
					m.On("GetOperation", mock.Anything, mock.Anything).Return(&opdomain.Operation{
						Name: "operations/123",
					}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				req: &longrunningpb.GetOperationRequest{Name: "operations/123"},
			},
			want: func(tt assert.TestingT, got interface{}, i2 ...interface{}) bool {
				operation, ok := got.(*longrunningpb.Operation)
				return assert.True(t, ok) && assert.Equal(t, "operations/123", operation.Name)
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockOperationService := mocks.NewOperationService(t)
			tt.fields.setupOperationService(mockOperationService)

			server := opapi.NewServer(mockOperationService)
			resp, err := server.GetOperation(tt.args.ctx, tt.args.req)

			tt.want(t, resp)
			tt.wantErr(t, err)

			mockOperationService.AssertExpectations(t)
		})
	}
}
