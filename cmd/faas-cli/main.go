package main

import (
	"context"
	"os/signal"
	"syscall"

	errorutils "github.com/10Narratives/faas/pkg/errors"
	"github.com/spf13/cobra"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rootCmd := &cobra.Command{
		Use:   "faas",
		Short: "Tool for serverless computing management",
		Long:  "Tool for managing serverless functions, operations, and related resources in the FaaS platform.",
	}

	rootCmd.AddCommand()

	errorutils.Try(rootCmd.ExecuteContext(ctx))
}
