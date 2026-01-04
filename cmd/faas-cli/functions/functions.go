package funccmd

import (
	"github.com/spf13/cobra"
)

func NewFunctionsGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "functions",
		Short: "Commands for managing serverless functions",
	}

	cmd.AddCommand()

	return cmd
}

func NewUploadFunctionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload function",
	}

	return cmd
}
