package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-events/common"
)

var rootCmd = &cobra.Command{
	Use:   common.RootCommand,
	Short: "Argo Events CLI",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(NewControllerCommand())
	rootCmd.AddCommand(NewEventSourceCommand())
	rootCmd.AddCommand(NewSensorCommand())
	rootCmd.AddCommand(NewWebhookCommand())
}
