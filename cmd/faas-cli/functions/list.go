package funccmd

import (
	"context"
	"fmt"
	"time"

	functionspb "github.com/10Narratives/faas/pkg/faas/v1/functions"
	"github.com/spf13/cobra"
)

func NewListFunctionsCmd() *cobra.Command {
	var (
		gatewayAddr string
		tls         bool
		caFile      string
		timeout     time.Duration

		pageSize  int32
		pageToken string
		all       bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List functions metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			if gatewayAddr == "" {
				return fmt.Errorf("--gateway is required")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			conn, err := dialGateway(ctx, gatewayAddr, tls, caFile)
			if err != nil {
				return err
			}
			defer conn.Close()

			client := functionspb.NewFunctionsClient(conn)

			printFn := func(fn *functionspb.Function) {
				uploadedAt := ""
				if ts := fn.GetUploadedAt(); ts != nil {
					uploadedAt = ts.AsTime().Format(time.RFC3339Nano)
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

				fmt.Fprintf(cmd.OutOrStdout(),
					"name=%s, display_name=%s, uploaded_at=%s, bundle_bucket=%s, bundle_object_key=%s, bundle_size=%d, bundle_sha256=%s\n",
					fn.GetName(),
					fn.GetDisplayName(),
					uploadedAt,
					bucket,
					objectKey,
					size,
					sha256hex,
				)
			}

			if !all {
				resp, err := client.ListFunctions(ctx, &functionspb.ListFunctionsRequest{
					PageSize:  pageSize,
					PageToken: pageToken,
				})
				if err != nil {
					return err
				}

				for _, fn := range resp.GetFunctions() {
					printFn(fn)
				}

				if t := resp.GetNextPageToken(); t != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "next_page_token=%s\n", t)
				}
				return nil
			}

			// --all: auto-pagination
			token := pageToken
			for {
				resp, err := client.ListFunctions(ctx, &functionspb.ListFunctionsRequest{
					PageSize:  pageSize,
					PageToken: token,
				})
				if err != nil {
					return err
				}

				for _, fn := range resp.GetFunctions() {
					printFn(fn)
				}

				if resp.GetNextPageToken() == "" {
					break
				}
				token = resp.GetNextPageToken()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&gatewayAddr, "gateway", "127.0.0.1:55055", "Gateway gRPC address host:port")
	cmd.Flags().BoolVar(&tls, "tls", false, "Use TLS")
	cmd.Flags().StringVar(&caFile, "tls-ca", "", "CA file (PEM), optional")
	cmd.Flags().DurationVar(&timeout, "timeout", 15*time.Second, "Overall timeout")

	cmd.Flags().Int32Var(&pageSize, "page-size", 50, "Max results per page (0 lets server decide)")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "Page token (from next_page_token)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages automatically")

	return cmd
}
