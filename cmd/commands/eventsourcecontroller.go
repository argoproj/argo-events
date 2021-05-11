package commands

import (
	"github.com/spf13/cobra"

	eventsourcecmd "github.com/argoproj/argo-events/controllers/eventsource/cmd"
	envpkg "github.com/argoproj/pkg/env"
)

func NewEventSourceControllerCommand() *cobra.Command {
	var (
		namespaced       bool
		managedNamespace string
	)

	command := &cobra.Command{
		Use:   "eventsource-controller",
		Short: "Start an EventSource controller",
		Run: func(cmd *cobra.Command, args []string) {
			eventsourcecmd.Start(namespaced, managedNamespace)
		},
	}
	command.Flags().BoolVar(&namespaced, "namespaced", false, "Whether to run in namespaced scope, defaults to false.")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", envpkg.LookupEnvStringOr("NAMESPACE", "argo-events"), "The namespace that the controller watches when \"--namespaced\" is \"true\".")
	return command
}
