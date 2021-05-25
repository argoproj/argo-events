package commands

import (
	"github.com/spf13/cobra"

	sensorcmd "github.com/argoproj/argo-events/controllers/sensor/cmd"
	envpkg "github.com/argoproj/pkg/env"
)

func NewSensorControllerCommand() *cobra.Command {
	var (
		namespaced       bool
		managedNamespace string
	)

	command := &cobra.Command{
		Use:   "sensor-controller",
		Short: "Start a Sensor controller",
		Run: func(cmd *cobra.Command, args []string) {
			sensorcmd.Start(namespaced, managedNamespace)
		},
	}
	command.Flags().BoolVar(&namespaced, "namespaced", false, "Whether to run in namespaced scope, defaults to false.")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", envpkg.LookupEnvStringOr("NAMESPACE", "argo-events"), "The namespace that the controller watches when \"--namespaced\" is \"true\".")
	return command
}
