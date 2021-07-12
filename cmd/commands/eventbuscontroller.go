package commands

import (
	"github.com/spf13/cobra"

	eventbuscmd "github.com/argoproj/argo-events/controllers/eventbus/cmd"
	envpkg "github.com/argoproj/pkg/env"
)

func NewEventBusControllerCommand() *cobra.Command {
	var (
		namespaced       bool
		managedNamespace string
	)

	command := &cobra.Command{
		Use:   "eventbus-controller",
		Short: "Start an EventBus controller",
		Run: func(cmd *cobra.Command, args []string) {
			eventbuscmd.Start(namespaced, managedNamespace)
		},
	}
	command.Flags().BoolVar(&namespaced, "namespaced", false, "Whether to run in namespaced scope, defaults to false.")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", envpkg.LookupEnvStringOr("NAMESPACE", "argo-events"), "The namespace that the controller watches when \"--namespaced\" is \"true\".")
	return command
}
