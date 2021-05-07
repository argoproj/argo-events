package commands

import (
	"github.com/spf13/cobra"

	eventsourcecmd "github.com/argoproj/argo-events/eventsources/cmd"
)

func NewEventSourceCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "eventsource-service",
		Short: "Start an event source service",
		Run: func(cmd *cobra.Command, args []string) {
			eventsourcecmd.Start()
		},
	}
	return command
}
