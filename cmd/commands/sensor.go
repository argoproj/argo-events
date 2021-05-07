package commands

import (
	"github.com/spf13/cobra"

	sensorcmd "github.com/argoproj/argo-events/sensors/cmd"
)

func NewSensorCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "sensor-service",
		Short: "Start sensor service",
		Run: func(cmd *cobra.Command, args []string) {
			sensorcmd.Start()
		},
	}
	return command
}
