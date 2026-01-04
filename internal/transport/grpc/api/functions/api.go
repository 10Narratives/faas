package funcapi

import (
	"bytes"
	"errors"
	"io"

	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	funcdomain "github.com/10Narratives/faas/internal/domain/functions"
	functionspb "github.com/10Narratives/faas/pkg/faas/functions/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxFunctionSourceSize = 1 << 20

type FunctionService interface {
	funcdomain.FunctionGetter
	funcdomain.FunctionSourceUploader
}

type Server struct {
	functionspb.UnimplementedFunctionServiceServer
	functionService FunctionService
}

func NewServer(functionService FunctionService) *Server {
	return &Server{
		functionService: functionService,
	}
}

func NewRegistration(functionService FunctionService) grpcsrv.ServiceRegistration {
	return func(s *grpc.Server) {
		functionspb.RegisterFunctionServiceServer(s, NewServer(functionService))
	}
}

func (s *Server) UploadFunctionSource(stream grpc.ClientStreamingServer[functionspb.UploadFunctionSourceRequest, functionspb.UploadFunctionSourceResponse]) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Error(codes.Unknown, "cannot receive upload function source metadata")
	}

	uploadMeta := req.GetMetadata()

	getFuncArgs := &funcdomain.GetFunctionArgs{Name: uploadMeta.GetFunctionName()}
	if _, err := s.functionService.GetFunction(stream.Context(), getFuncArgs); err != nil {
		if !errors.Is(err, funcdomain.ErrFunctionNotFound) {
			return status.Error(codes.AlreadyExists, "function with received name already exists")
		}
	}

	funcSrcData := &bytes.Buffer{}
	funcSrcSize := 0

	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return status.Error(codes.Unknown, "cannot receive function source chunk")
		}

		chunkData := req.GetChunk().GetData()

		funcSrcSize += len(chunkData)
		if funcSrcSize > maxFunctionSourceSize {
			return status.Error(codes.InvalidArgument, "function source data is too long")
		}

		if _, err := funcSrcData.Write(chunkData); err != nil {
			return status.Error(codes.Internal, "cannot upload function source")
		}
	}

	uploadFuncSrcArgs := &funcdomain.UploadFunctionSourceArgs{
		Metadata: funcdomain.UploadFunctionSourceMetadata{
			FunctionName: uploadMeta.GetFunctionName(),
			SourceBundleMetadata: funcdomain.SourceBundleMetadata{
				Type:     funcdomain.SourceBundleType(uploadMeta.GetSourceBundleMetadata().GetType()),
				FileName: uploadMeta.GetSourceBundleMetadata().GetFileName(),
				Size:     uploadMeta.GetSourceBundleMetadata().GetSizeBytes(),
			},
		},
	}

	uploadRes, err := s.functionService.UploadFunctionSource(stream.Context(), uploadFuncSrcArgs)
	if err != nil {
		return status.Error(codes.Internal, "cannot upload function source")
	}

	return stream.SendAndClose(&functionspb.UploadFunctionSourceResponse{
		FunctionId: uploadRes.FunctionID,
		ObjectKey:  uploadRes.ObjectKey,
	})
}
