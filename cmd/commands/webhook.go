package commands

import (
	"github.com/spf13/cobra"

	webhookcmd "github.com/argoproj/argo-events/webhook/cmd"
)

func NewWebhookCommand() *cobra.Command {
	var (
		namespaced bool
	)

	command := &cobra.Command{
		Use:   "webhook-service",
		Short: "Start validating webhook server",
		Run: func(cmd *cobra.Command, args []string) {
			webhookcmd.Start(namespaced)
		},
	}

	command.Flags().BoolVar(&namespaced, "namespaced", false, "Whether to run in namespaced scope, defaults to false.")
	return command
}
