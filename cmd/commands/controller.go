package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	eventbuscmd "github.com/argoproj/argo-events/controllers/eventbus/cmd"
	eventsourcecmd "github.com/argoproj/argo-events/controllers/eventsource/cmd"
	sensorcmd "github.com/argoproj/argo-events/controllers/sensor/cmd"
	envpkg "github.com/argoproj/pkg/env"
)

func NewControllerCommand() *cobra.Command {
	var (
		component        string
		namespaced       bool
		managedNamespace string
	)

	command := &cobra.Command{
		Use:   "controller",
		Short: "Start a controller",
		Run: func(cmd *cobra.Command, args []string) {
			switch component {
			case "eventbus":
				eventbuscmd.Start(namespaced, managedNamespace)
			case "eventsource":
				eventsourcecmd.Start(namespaced, managedNamespace)
			case "sensor":
				sensorcmd.Start(namespaced, managedNamespace)
			case "":
				cmd.HelpFunc()(cmd, args)
			default:
				fmt.Printf("Error: Invalid component: %s\n\n", component)
				cmd.HelpFunc()(cmd, args)
			}
		},
	}
	defaultNamespace := envpkg.LookupEnvStringOr("NAMESPACE", "argo-events")
	command.Flags().StringVar(&component, "component", "", `The controller component, possible values: "eventbus", "eventsource" or "sensor".`)
	command.Flags().BoolVar(&namespaced, "namespaced", false, "Whether to run in namespaced scope, defaults to false.")
	command.Flags().StringVar(&managedNamespace, "managed-namespaces", defaultNamespace, "The namespace that the controller watches when \"--namespaced\" is \"true\".")
	return command
}
