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
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
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
	eventType := apicommon.EventSourceType(args[0])
	es, err := getEventingServer(eventType, restConfig, clientset, namespace, common.NewArgoEventsLogger())
	if err != nil {
		panic(err)
	}
	server.StartGateway(es)
}

func getEventingServer(eventType apicommon.EventSourceType, restConfig *rest.Config, clientset kubernetes.Interface, namespace string, log *logrus.Logger) (gateways.EventingServer, error) {
	switch eventType {
	case apicommon.AMQPEvent:
		return &amqp.EventListener{Logger: log}, nil
	case apicommon.SNSEvent:
		return &aws_sns.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.SQSEvent:
		return &aws_sqs.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.AzureEventsHub:
		return &azure_events_hub.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.CalendarEvent:
		return &calendar.EventListener{Logger: log}, nil
	case apicommon.EmitterEvent:
		return &emitter.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.FileEvent:
		return &file.EventListener{Logger: log}, nil
	case apicommon.PubSubEvent:
		return &pubsub.EventListener{Logger: log}, nil
	case apicommon.GitHubEvent:
		return &github.EventListener{Logger: log, Namespace: namespace, K8sClient: clientset}, nil
	case apicommon.GitLabEvent:
		return &gitlab.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.HDFSEvent:
		return &hdfs.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.KafkaEvent:
		return &kafka.EventListener{Logger: log}, nil
	case apicommon.MinioEvent:
		return &minio.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.MQTTEvent:
		return &mqtt.EventListener{Logger: log}, nil
	case apicommon.NATSEvent:
		return &nats.EventListener{Logger: log}, nil
	case apicommon.NSQEvent:
		return &nsq.EventListener{Logger: log}, nil
	case apicommon.RedisEvent:
		return &redis.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.ResourceEvent:
		return &resource.EventListener{Logger: log, K8RestConfig: restConfig}, nil
	case apicommon.SlackEvent:
		return &slack.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.StorageGridEvent:
		return &storagegrid.EventListener{Logger: log}, nil
	case apicommon.StripeEvent:
		return &stripe.EventListener{Logger: log, K8sClient: clientset, Namespace: namespace}, nil
	case apicommon.WebhookEvent:
		return &webhook.EventListener{Logger: log}, nil
	default:
		return nil, errors.New("invalid event type")
	}
}
