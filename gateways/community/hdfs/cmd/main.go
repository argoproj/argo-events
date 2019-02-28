package main

import (
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/community/hdfs"
	"k8s.io/client-go/kubernetes"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)
	namespace, ok := os.LookupEnv(common.EnvVarGatewayNamespace)
	if !ok {
		panic("namespace is not provided")
	}
	gateways.StartGateway(&hdfs.EventSourceExecutor{
		Log:       common.GetLoggerContext(common.LoggerConf()).Logger(),
		Namespace: namespace,
		Clientset: clientset,
	})
}
