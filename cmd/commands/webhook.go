package commands

import (
	"github.com/spf13/cobra"

	webhookcmd "github.com/argoproj/argo-events/webhook/cmd"
)

func NewWebhookCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "webhook-service",
		Short: "Start validating webhook server",
		Run: func(cmd *cobra.Command, args []string) {
			webhookcmd.Start()
		},
	}
	return command
}
