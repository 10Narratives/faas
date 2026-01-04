package funccmd

import (
	"github.com/spf13/cobra"
)

func NewFunctionsGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "functions",
		Short: "Commands for managing serverless functions",
	}

	cmd.AddCommand(
		NewUploadFunctionCommand(),
	)

	return cmd
}
