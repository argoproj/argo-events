package main

import (
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/gateways/server/amqp"
	aws_sns "github.com/argoproj/argo-events/gateways/server/aws-sns"
	aws_sqs "github.com/argoproj/argo-events/gateways/server/aws-sqs"
	azure_events_hub "github.com/argoproj/argo-events/gateways/server/azure-events-hub"
	"github.com/argoproj/argo-events/gateways/server/calendar"
	"github.com/argoproj/argo-events/gateways/server/emitter"
	"github.com/argoproj/argo-events/gateways/server/file"
	pubsub "github.com/argoproj/argo-events/gateways/server/gcp-pubsub"
	"github.com/argoproj/argo-events/gateways/server/github"
	"github.com/argoproj/argo-events/gateways/server/gitlab"
	"github.com/argoproj/argo-events/gateways/server/hdfs"
	"github.com/argoproj/argo-events/gateways/server/kafka"
	"github.com/argoproj/argo-events/gateways/server/minio"
	"github.com/argoproj/argo-events/gateways/server/mqtt"
	"github.com/argoproj/argo-events/gateways/server/nats"
	"github.com/argoproj/argo-events/gateways/server/nsq"
	"github.com/argoproj/argo-events/gateways/server/redis"
	"github.com/argoproj/argo-events/gateways/server/resource"
	"github.com/argoproj/argo-events/gateways/server/slack"
	"github.com/argoproj/argo-events/gateways/server/storagegrid"
	"github.com/argoproj/argo-events/gateways/server/stripe"
	"github.com/argoproj/argo-events/gateways/server/webhook"
	gatewayv1alpha1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		panic("gateway type argument is not provided")
	}
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		panic("namespace is not provided")
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)
	eventType := gatewayv1alpha1.EventSourceType(args[0])
	es, err := getEventingServer(eventType, restConfig, clientset, namespace, common.NewArgoEventsLogger())
	if err != nil {
		panic(err)
	}
	server.StartGateway(es)
}

func getEventingServer(eventType gatewayv1alpha1.EventSourceType, restConfig *rest.Config, clientset kubernetes.Interface, namespace string, log *logrus.Logger) (gateways.EventingServer, error) {
	switch eventType {
	case gatewayv1alpha1.AMQPEvent:
		return &amqp.EventListener{Logger: log}, nil
	case gatewayv1alpha1.SNSEvent:
		return &aws_sns.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.SQSEvent:
		return &aws_sqs.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.AzureEventsHub:
		return &azure_events_hub.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.CalendarEvent:
		return &calendar.EventListener{Logger: log}, nil
	case gatewayv1alpha1.EmitterEvent:
		return &emitter.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.FileEvent:
		return &file.EventListener{Logger: log}, nil
	case gatewayv1alpha1.PubSubEvent:
		return &pubsub.EventListener{Logger: log}, nil
	case gatewayv1alpha1.GitHubEvent:
		return &github.EventListener{Logger: log, Namespace: namespace, K8sClient: clientset}, nil
	case gatewayv1alpha1.GitLabEvent:
		return &gitlab.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.HDFSEvent:
		return &hdfs.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.KafkaEvent:
		return &kafka.EventListener{Logger: log}, nil
	case gatewayv1alpha1.MinioEvent:
		return &minio.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.MQTTEvent:
		return &mqtt.EventListener{Logger: log}, nil
	case gatewayv1alpha1.NATSEvent:
		return &nats.EventListener{Logger: log}, nil
	case gatewayv1alpha1.NSQEvent:
		return &nsq.EventListener{Logger: log}, nil
	case gatewayv1alpha1.RedisEvent:
		return &redis.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.ResourceEvent:
		return &resource.EventListener{Logger: log, K8RestConfig: restConfig}, nil
	case gatewayv1alpha1.SlackEvent:
		return &slack.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.StorageGridEvent:
		return &storagegrid.EventListener{Logger: log}, nil
	case gatewayv1alpha1.StripeEvent:
		return &stripe.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case gatewayv1alpha1.WebhookEvent:
		return &webhook.EventListener{Logger: log}, nil
	default:
		return nil, errors.New("invalid event type")
	}
}
