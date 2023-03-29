package commands

import (
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	controllercmd "github.com/argoproj/argo-events/controllers/cmd"
	envpkg "github.com/argoproj/pkg/env"
)

func NewControllerCommand() *cobra.Command {
	var (
		leaderElection   bool
		namespaced       bool
		managedNamespace string
		metricsPort      int32
		healthPort       int32
		klogLevel        int
	)

	command := &cobra.Command{
		Use:   "controller",
		Short: "Start the controller",
		Run: func(cmd *cobra.Command, args []string) {
			logging.SetKlogLevel(klogLevel)
			eventOpts := controllercmd.ArgoEventsControllerOpts{
				LeaderElection:   leaderElection,
				ManagedNamespace: managedNamespace,
				Namespaced:       namespaced,
				MetricsPort:      metricsPort,
				HealthPort:       healthPort,
			}
			controllercmd.Start(eventOpts)
		},
	}
	command.Flags().BoolVar(&namespaced, "namespaced", false, "Whether to run in namespaced scope, defaults to false.")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", envpkg.LookupEnvStringOr("NAMESPACE", "argo-events"), "The namespace that the controller watches when \"--namespaced\" is \"true\".")
	command.Flags().BoolVar(&leaderElection, "leader-election", true, "Enable leader election")
	command.Flags().Int32Var(&metricsPort, "metrics-port", common.ControllerMetricsPort, "Metrics port")
	command.Flags().Int32Var(&healthPort, "health-port", common.ControllerHealthPort, "Health port")
	command.Flags().IntVar(&klogLevel, "kloglevel", 0, "klog level")
	return command
}
