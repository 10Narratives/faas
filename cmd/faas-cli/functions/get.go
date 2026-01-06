package funccmd

import (
	"context"
	"fmt"
	"time"

	functionspb "github.com/10Narratives/faas/pkg/faas/v1/functions"
	"github.com/spf13/cobra"
)

func NewGetFunctionCmd() *cobra.Command {
	var (
		functionName string
		gatewayAddr  string
		tls          bool
		caFile       string
		timeout      time.Duration
	)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get function metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			if functionName == "" {
				return fmt.Errorf("--name is required")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			conn, err := dialGateway(ctx, gatewayAddr, tls, caFile)
			if err != nil {
				return err
			}
			defer conn.Close()

			client := functionspb.NewFunctionsClient(conn)
			fn, err := client.GetFunction(ctx, &functionspb.GetFunctionRequest{
				Name: functionName,
			})
			if err != nil {
				return err
			}

			var (
				bucket    string
				objectKey string
				size      uint64
				sha256hex string
			)
			if sb := fn.GetSourceBundle(); sb != nil {
				bucket = sb.GetBucket()
				objectKey = sb.GetObjectKey()
				size = sb.GetSize()
				sha256hex = sb.GetSha256()
			}

			uploadedAt := ""
			if ts := fn.GetUploadedAt(); ts != nil {
				uploadedAt = ts.AsTime().Format(time.RFC3339Nano)
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"function: name=%s, display_name=%s, uploaded_at=%s, bundle_bucket=%s, bundle_object_key=%s, bundle_size=%d, bundle_sha256=%s\n",
				fn.GetName(),
				fn.GetDisplayName(),
				uploadedAt,
				bucket,
				objectKey,
				size,
				sha256hex,
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&functionName, "name", "", "Function name, e.g. functions/my-func")
	cmd.Flags().StringVar(&gatewayAddr, "gateway", "127.0.0.1:55055", "Gateway gRPC address host:port")
	cmd.Flags().BoolVar(&tls, "tls", false, "Use TLS")
	cmd.Flags().StringVar(&caFile, "tls-ca", "", "CA file (PEM), optional")
	cmd.Flags().DurationVar(&timeout, "timeout", 15*time.Second, "Overall timeout")

	return cmd
}
