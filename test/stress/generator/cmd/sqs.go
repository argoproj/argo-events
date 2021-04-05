package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func NewSqsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "sqs",
		Short: "Generate SQS messages",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			validateGlobalParameters(cmd, args)

			stressRun(signals.SetupSignalHandler(), func() error {
				return fmt.Errorf("To be implemented.")
			})
		},
	}

	return command
}
