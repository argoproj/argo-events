package main

import (
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core/hdfs"
	"k8s.io/client-go/kubernetes"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gateways.StartGateway(&hdfs.EventListener{
		Logger:    common.NewArgoEventsLogger(),
		K8sClient: clientset,
	})
}
