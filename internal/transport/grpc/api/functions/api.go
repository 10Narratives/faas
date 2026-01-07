package funcapi

import (
	"context"
	"errors"
	"io"

	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	funcdomain "github.com/10Narratives/faas/internal/domains/functions"
	faaspb "github.com/10Narratives/faas/pkg/faas/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate mockery --name FunctionService --output ./mocks --outpkg mocks --with-expecter --filename function_service.go
type FunctionService interface {
	funcdomain.FunctionUploader
	funcdomain.FunctionExecutor
	funcdomain.FunctionGetter
	funcdomain.FunctionLister
	funcdomain.FunctionDeleter
}

type Server struct {
	faaspb.UnimplementedFunctionsServer
	functionService FunctionService
}

func NewServer(functionService FunctionService) *Server {
	return &Server{functionService: functionService}
}

func NewRegistration(functionService FunctionService) grpcsrv.ServiceRegistration {
	return func(s *grpc.Server) {
		faaspb.RegisterFunctionsServer(s, NewServer(functionService))
	}
}

func (s *Server) UploadFunction(stream grpc.ClientStreamingServer[faaspb.UploadFunctionRequest, faaspb.Function]) error {
	ctx := stream.Context()

	first, err := stream.Recv()
	if err == io.EOF {
		return status.Error(codes.InvalidArgument, "missing upload metadata")
	}
	if err != nil {
		return toStatusErr(err)
	}

	meta := first.GetUploadFunctionMetadata()
	if meta == nil {
		return status.Error(codes.InvalidArgument, "first message must be upload_function_metadata")
	}

	name, err := funcdomain.ParseFunctionName(meta.FunctionName)
	if err != nil {
		return toStatusErr(err)
	}

	format, err := pbToDomainUploadFormat(meta.Format)
	if err != nil {
		return toStatusErr(err)
	}

	pr, pw := io.Pipe()

	type uploadResult struct {
		res *funcdomain.UploadFunctionResult
		err error
	}
	done := make(chan uploadResult, 1)

	go func() {
		res, uerr := s.functionService.UploadFunction(ctx, &funcdomain.UploadFunctionArgs{
			Name:   name,
			Format: format,
			Data:   pr,
		})
		_ = pr.Close()
		done <- uploadResult{res: res, err: uerr}
	}()

	for {
		req, rerr := stream.Recv()
		if rerr == io.EOF {
			_ = pw.Close()
			break
		}
		if rerr != nil {
			_ = pw.CloseWithError(rerr)
			<-done
			return toStatusErr(rerr)
		}

		if req.GetUploadFunctionMetadata() != nil {
			_ = pw.CloseWithError(funcdomain.ErrInvalidArgument)
			<-done
			return status.Error(codes.InvalidArgument, "upload_function_metadata must be sent only once (first message)")
		}

		chunk := req.GetUploadFunctionData()
		if chunk == nil {
			_ = pw.CloseWithError(funcdomain.ErrInvalidArgument)
			<-done
			return status.Error(codes.InvalidArgument, "expected upload_function_data")
		}

		if len(chunk.Data) > 0 {
			if _, werr := pw.Write(chunk.Data); werr != nil {
				_ = pw.CloseWithError(werr)
				<-done
				return toStatusErr(werr)
			}
		}
	}

	ur := <-done
	if ur.err != nil {
		return toStatusErr(ur.err)
	}
	if ur.res == nil || ur.res.Function == nil {
		return status.Error(codes.Internal, "upload finished without function result")
	}

	return stream.SendAndClose(domainToPBFunction(ur.res.Function))
}

func (s *Server) ExecuteFunction(ctx context.Context, req *faaspb.ExecuteFunctionRequest) (*faaspb.ExecuteFunctionResponse, error) {
	name, err := funcdomain.ParseFunctionName(req.GetName())
	if err != nil {
		return nil, toStatusErr(err)
	}

	res, err := s.functionService.ExecuteFunction(ctx, &funcdomain.ExecuteFunctionArgs{
		Name:       name,
		Parameters: req.GetParameters(),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	if res == nil {
		return nil, status.Error(codes.Internal, "missing execute function result")
	}

	outName := res.TaskName
	if outName == "" {
		outName = string(name)
	}

	return &faaspb.ExecuteFunctionResponse{
		Name: outName,
	}, nil
}

func (s *Server) GetFunction(ctx context.Context, req *faaspb.GetFunctionRequest) (*faaspb.Function, error) {
	name, err := funcdomain.ParseFunctionName(req.GetName())
	if err != nil {
		return nil, toStatusErr(err)
	}

	res, err := s.functionService.GetFunction(ctx, &funcdomain.GetFunctionArgs{Name: name})
	if err != nil {
		return nil, toStatusErr(err)
	}
	if res == nil || res.Function == nil {
		return nil, status.Error(codes.Internal, "missing function in result")
	}

	return domainToPBFunction(res.Function), nil
}

func (s *Server) ListFunctions(ctx context.Context, req *faaspb.ListFunctionsRequest) (*faaspb.ListFunctionsResponse, error) {
	res, err := s.functionService.ListFunctions(ctx, &funcdomain.ListFunctionsArgs{
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}

	out := &faaspb.ListFunctionsResponse{
		Functions:     make([]*faaspb.Function, 0, len(res.Functions)),
		NextPageToken: res.NextPageToken,
	}

	for _, f := range res.Functions {
		if f == nil {
			continue
		}
		out.Functions = append(out.Functions, domainToPBFunction(f))
	}

	return out, nil
}

func (s *Server) DeleteFunction(ctx context.Context, req *faaspb.DeleteFunctionRequest) (*emptypb.Empty, error) {
	name, err := funcdomain.ParseFunctionName(req.GetName())
	if err != nil {
		return nil, toStatusErr(err)
	}

	if err := s.functionService.DeleteFunction(ctx, &funcdomain.DeleteFunctionArgs{Name: name}); err != nil {
		return nil, toStatusErr(err)
	}

	return &emptypb.Empty{}, nil
}

func pbToDomainUploadFormat(f faaspb.UploadFunctionMetadata_Format) (funcdomain.UploadFunctionFormat, error) {
	switch f {
	case faaspb.UploadFunctionMetadata_FORMAT_ZIP:
		return funcdomain.ZipFormat, nil
	case faaspb.UploadFunctionMetadata_FORMAT_TAR_GZ:
		return funcdomain.TarGZFormat, nil
	default:
		return "", funcdomain.ErrInvalidArgument
	}
}

func domainToPBFunction(f *funcdomain.Function) *faaspb.Function {
	pb := &faaspb.Function{
		Name:        string(f.Name),
		DisplayName: f.DisplayName,
		UploadedAt:  timestamppb.New(f.UploadedAt),
		SourceBundle: &faaspb.SourceBundle{
			Bucket:    f.Bundle.Bucket,
			ObjectKey: f.Bundle.ObjectKey,
			Size:      f.Bundle.Size,
			Sha256:    f.Bundle.SHA256,
		},
	}
	return pb
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
	case errors.Is(err, funcdomain.ErrFunctionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, funcdomain.ErrFunctionAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, funcdomain.ErrInvalidArgument),
		errors.Is(err, funcdomain.ErrInvalidName),
		errors.Is(err, funcdomain.ErrInvalidPageToken),
		errors.Is(err, funcdomain.ErrUnsupportedFormat):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
