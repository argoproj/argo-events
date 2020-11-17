package main

import (
	"github.com/blushft/go-diagrams/diagram"
	"github.com/blushft/go-diagrams/nodes/apps"
	"github.com/blushft/go-diagrams/nodes/k8s"
	"github.com/blushft/go-diagrams/nodes/oci"
	"github.com/blushft/go-diagrams/nodes/programming"
)

func main() {
	d, err := diagram.New(diagram.Filename("diagram"), diagram.Label("Argo Events"), diagram.Direction("LR"))
	if err != nil {
		panic(err)
	}

	kubeCluster := diagram.NewGroup("kubernetes-cluster").Label("Kubernetes Cluster")

	k8sapi := k8s.Controlplane.Api(diagram.NodeLabel("Kubernetes API"))

	kubeCluster.NewGroup("kube-system").
		Label("kube-system namespace").
		Add(k8sapi)

	eventSourceController := k8s.Compute.Pod(diagram.NodeLabel("Event Source Controller"))
	sensorController := k8s.Compute.Pod(diagram.NodeLabel("Sensor Controller"))
	eventBusController := k8s.Compute.Pod(diagram.NodeLabel("Event Bus Controller"))

	kubeCluster.NewGroup("argo-events").
		Label("argo system namespace").
		Add(sensorController, eventSourceController, eventBusController)

	d.Connect(eventSourceController, k8sapi).Group(kubeCluster)
	d.Connect(sensorController, k8sapi).Group(kubeCluster)
	d.Connect(eventBusController, k8sapi).Group(kubeCluster)

	eventSource := k8s.Compute.Pod(diagram.NodeLabel("Event Source"))
	eventBus := k8s.Compute.Sts(diagram.NodeLabel("Event Bus"))
	sensor := k8s.Compute.Pod(diagram.NodeLabel("Sensor"))

	user := apps.Client.User(diagram.NodeLabel("User"))
	kubectl := programming.Language.Bash(diagram.NodeLabel("Kubectl CLI"))

	d.Connect(kubectl, k8sapi)
	d.Connect(user, kubectl)

	kubeCluster.NewGroup("user namespace").
		Label("user namespace ").
		Add(eventSource, eventBus, sensor).
		Connect(eventSource, eventBus).
		Connect(eventBus, sensor)

	for _, x := range []*diagram.Node{
		oci.Network.LoadBalancer(diagram.NodeLabel("Webhook / Stripe")),
		oci.Monitoring.Alarm(diagram.NodeLabel("Cron Schedule")),
		apps.Vcs.Git(diagram.NodeLabel("Github / Gitlab")),
		oci.Storage.FileStorage(diagram.NodeLabel("File: e.g. HDFS, AWS S3")),
		apps.Queue.Kafka(diagram.NodeLabel("Queue: e.g. Kafka, NATS, GCP PubSub")),
		oci.Compute.Container(diagram.NodeLabel("Custom")),
	} {
		d.Connect(x, eventSource)
	}

	for _, x := range []*diagram.Node{
		oci.Compute.Functions(diagram.NodeLabel("Function: AWS Lambda / Apache OpenWhisk ")),
		apps.Queue.Kafka(diagram.NodeLabel("Queue: Kafka / NATS")),
		apps.Gitops.Argocd(diagram.NodeLabel("Argo Workflows")),
		k8s.Others.Crd(diagram.NodeLabel("Kubernetes Resource")),
		oci.Network.LoadBalancer(diagram.NodeLabel("HTTP Request")),
		oci.Compute.Container(diagram.NodeLabel("Custom")),
	} {
		d.Connect(sensor, x)
	}

	if err := d.Render(); err != nil {
		panic(err)
	}
}
