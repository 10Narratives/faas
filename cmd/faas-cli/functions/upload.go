package funccmd

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	functionspb "github.com/10Narratives/faas/pkg/faas/v1/functions"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func NewUploadFunctionCmd() *cobra.Command {
	var (
		functionName string
		srcDir       string
		gatewayAddr  string
		format       string
		tls          bool
		caFile       string
		timeout      time.Duration
	)

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Archive user code and upload it",
		RunE: func(cmd *cobra.Command, args []string) error {
			if functionName == "" {
				return fmt.Errorf("--name is required")
			}
			if srcDir == "" {
				return fmt.Errorf("--path is required")
			}

			absSrc, err := filepath.Abs(srcDir)
			if err != nil {
				return err
			}

			tmpDir := os.TempDir()
			archivePath := filepath.Join(tmpDir, fmt.Sprintf("faas-%d.zip", time.Now().UnixNano()))

			switch format {
			case "zip", "":
				if err := zipDir(absSrc, archivePath); err != nil {
					return fmt.Errorf("zipDir: %w", err)
				}
			default:
				return fmt.Errorf("unsupported --format=%q (implemented: zip)", format)
			}
			defer os.Remove(archivePath)

			sha, size, err := fileSHA256AndSize(archivePath)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			conn, err := dialGateway(ctx, gatewayAddr, tls, caFile)
			if err != nil {
				return err
			}
			defer conn.Close()

			client := functionspb.NewFunctionsClient(conn)
			fn, err := uploadArchive(ctx, client, functionName, archivePath, functionspb.UploadFunctionMetadata_FORMAT_ZIP)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"uploaded: name=%s, local_archive_size=%d, local_sha256=%s, bundle_bucket=%s, bundle_object_key=%s\n",
				fn.GetName(), size, sha, fn.GetSourceBundle().GetBucket(), fn.GetSourceBundle().GetObjectKey(),
			)
			return nil
		},
	}

	cmd.Flags().StringVar(&functionName, "name", "", "Function name, e.g. functions/my-func")
	cmd.Flags().StringVar(&srcDir, "path", "", "Path to user code directory")
	cmd.Flags().StringVar(&gatewayAddr, "gateway", "127.0.0.1:55055", "Gateway gRPC address host:port")
	cmd.Flags().StringVar(&format, "format", "zip", "Archive format: zip")
	cmd.Flags().BoolVar(&tls, "tls", false, "Use TLS")
	cmd.Flags().StringVar(&caFile, "tls-ca", "", "CA file (PEM), optional")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Overall timeout")

	return cmd
}

func dialGateway(ctx context.Context, addr string, useTLS bool, caFile string) (*grpc.ClientConn, error) {
	var creds credentials.TransportCredentials
	if useTLS {
		if caFile != "" {
			c, err := credentials.NewClientTLSFromFile(caFile, "")
			if err != nil {
				return nil, err
			}
			creds = c
		} else {
			creds = credentials.NewTLS(nil)
		}
	} else {
		creds = insecure.NewCredentials()
	}

	return grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(creds))
}

func uploadArchive(
	ctx context.Context,
	client functionspb.FunctionsClient,
	functionName string,
	archivePath string,
	format functionspb.UploadFunctionMetadata_Format,
) (*functionspb.Function, error) {
	stream, err := client.UploadFunction(ctx)
	if err != nil {
		return nil, err
	}

	if err := stream.Send(&functionspb.UploadFunctionRequest{
		Payload: &functionspb.UploadFunctionRequest_UploadFunctionMetadata{
			UploadFunctionMetadata: &functionspb.UploadFunctionMetadata{
				FunctionName: functionName,
				Format:       format,
			},
		},
	}); err != nil {
		return nil, err
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	const chunkSize = 1 << 20
	buf := make([]byte, chunkSize)

	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if err := stream.Send(&functionspb.UploadFunctionRequest{
				Payload: &functionspb.UploadFunctionRequest_UploadFunctionData{
					UploadFunctionData: &functionspb.UploadFunctionData{
						Data: buf[:n],
					},
				},
			}); err != nil {
				return nil, err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}

	// 3) server response
	return stream.CloseAndRecv()
}

func zipDir(srcDir, dstZip string) error {
	out, err := os.Create(dstZip)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = rel
		h.Method = zip.Deflate

		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		_, err = io.Copy(w, in)
		return err
	})
}

func fileSHA256AndSize(path string) (shaHex string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
