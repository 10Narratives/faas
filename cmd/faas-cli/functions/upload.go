package funccmd

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	funcdomain "github.com/10Narratives/faas/internal/domain/functions"
	functionspb "github.com/10Narratives/faas/pkg/faas/functions/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

func NewUploadFunctionCommand() *cobra.Command {
	var manifestPath string
	var outPath string
	var addr string

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Archive sources from manifest upload.source_dir",
		RunE: func(cmd *cobra.Command, args []string) error {
			absManifestPath, err := filepath.Abs(manifestPath)
			if err != nil {
				return fmt.Errorf("abs manifest path: %w", err)
			}

			b, err := os.ReadFile(absManifestPath)
			if err != nil {
				return fmt.Errorf("read manifest: %w", err)
			}

			var m funcdomain.Manifest
			if err := yaml.Unmarshal(b, &m); err != nil {
				return fmt.Errorf("parse manifest yaml: %w", err)
			}

			if m.Upload.SourceDir == "" {
				return fmt.Errorf("manifest upload.source_dir is empty")
			}

			manifestDir := filepath.Dir(absManifestPath)

			sourceDir := m.Upload.SourceDir
			if !filepath.IsAbs(sourceDir) {
				sourceDir = filepath.Join(manifestDir, sourceDir)
			}

			if outPath == "" {
				switch {
				case m.Name != "":
					outPath = m.Name + ".zip"
				default:
					outPath = "archive.zip"
				}
			}

			if err := zipDir(sourceDir, outPath); err != nil {
				return err
			}

			resp, err := uploadZip(cmd.Context(), addr, m.Name, outPath)
			if err != nil {
				return err
			}

			fmt.Println("response", resp)

			return nil
		},
	}

	cmd.Flags().StringVarP(&manifestPath, "manifest", "m", "manifest.yaml", "Path to manifest YAML")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "Output archive path (default: <name>.zip)")
	cmd.Flags().StringVarP(&addr, "addr", "a", "", "Address of faas gateway")

	return cmd
}

func zipDir(sourceDir, outZip string) error {
	srcAbs, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("abs source dir: %w", err)
	}

	f, err := os.Create(outZip)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	return filepath.WalkDir(srcAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return err
		}

		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(rel)
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

func uploadZip(ctx context.Context, addr, functionName, zipPath string) (*functionspb.UploadFunctionSourceResponse, error) {
	st, err := os.Stat(zipPath)
	if err != nil {
		return nil, fmt.Errorf("stat zip: %w", err)
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}
	defer conn.Close()

	client := functionspb.NewFunctionServiceClient(conn)

	stream, err := client.UploadFunctionSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("open upload stream: %w", err)
	}

	md := &functionspb.UploadFunctionSourceRequest{
		Payload: &functionspb.UploadFunctionSourceRequest_Metadata{
			Metadata: &functionspb.UploadFunctionSourceMetadata{
				FunctionName: functionName,
				SourceBundleMetadata: &functionspb.SourceBundleMetadata{
					Type:      functionspb.SourceBundleMetadata_BUNDLE_TYPE_ZIP,
					FileName:  filepath.Base(zipPath),
					SizeBytes: uint64(st.Size()),
				},
			},
		},
	}
	if err := stream.Send(md); err != nil {
		return nil, fmt.Errorf("send metadata: %w", err)
	}

	f, err := os.Open(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer f.Close()

	const chunkSize = 64 * 1024
	r := bufio.NewReader(f)
	buf := make([]byte, chunkSize)

	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read zip chunk: %w", err)
		}

		req := &functionspb.UploadFunctionSourceRequest{
			Payload: &functionspb.UploadFunctionSourceRequest_Chunk{
				Chunk: &functionspb.UploadChunk{
					Data: buf[:n],
				},
			},
		}

		if err := stream.Send(req); err != nil {
			return nil, fmt.Errorf("send chunk: %w", err)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("close and recv: %w", err)
	}
	return res, nil
}
