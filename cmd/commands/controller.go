package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-events/common"
	eventbuscmd "github.com/argoproj/argo-events/controllers/eventbus/cmd"
	eventsourcecmd "github.com/argoproj/argo-events/controllers/eventsource/cmd"
	sensorcmd "github.com/argoproj/argo-events/controllers/sensor/cmd"
	envpkg "github.com/argoproj/pkg/env"
)

const (
	componentValEventBus    = "eventbus"
	componentValEventSource = "eventsource"
	componentValSensor      = "sensor"
)

func NewControllerCommand() *cobra.Command {
	var (
		component        string
		namespaced       bool
		managedNamespace string
	)

	command := &cobra.Command{
		Use:   common.ControllerCommand,
		Short: "Start a controller",
		Run: func(cmd *cobra.Command, args []string) {
			switch component {
			case componentValEventBus:
				eventbuscmd.Start(namespaced, managedNamespace)
			case componentValEventSource:
				eventsourcecmd.Start(namespaced, managedNamespace)
			case componentValSensor:
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
	command.Flags().StringVar(&component, "component", "", `The controller component, possible values: "`+componentValEventBus+`", "`+componentValEventSource+`" or "`+componentValSensor+`".`)
	command.Flags().BoolVar(&namespaced, "namespaced", false, "Whether to run in namespaced scope, defaults to false.")
	command.Flags().StringVar(&managedNamespace, "managed-namespaces", defaultNamespace, "The namespace that the controller watches when \"--namespaced\" is \"true\".")
	return command
}
