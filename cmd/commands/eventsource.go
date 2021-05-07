package commands

import (
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-events/common"
	eventsourcecmd "github.com/argoproj/argo-events/eventsources/cmd"
)

func NewEventSourceCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   common.EventSourceSvcCommand,
		Short: "Start event source service",
		Run: func(cmd *cobra.Command, args []string) {
			eventsourcecmd.Start()
		},
	}
	return command
}
