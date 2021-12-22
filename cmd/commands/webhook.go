package commands

import (
	"github.com/spf13/cobra"

	webhookcmd "github.com/argoproj/argo-events/webhook/cmd"
)

func NewWebhookCommand() *cobra.Command {
	var (
		namespaced       bool
		namespacedKey    string
		namespacedValues string
	)

	command := &cobra.Command{
		Use:   "webhook-service",
		Short: "Start validating webhook server",
		Run: func(cmd *cobra.Command, args []string) {
			webhookcmd.Start(namespaced, namespacedKey, namespacedValues)
		},
	}

	command.Flags().BoolVar(&namespaced, "namespaced", false, "Whether to run in namespaced scope, defaults to false.")
	command.Flags().StringVar(&namespacedKey, "namespacedKey", "", "If namespaced is true, a key is required for the matchExpression")
	command.Flags().StringVar(&namespacedValues, "namespacedValues", "", "If namespaced is true, a comma separated array is required for the matchExpression")
	return command
}
