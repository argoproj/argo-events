package tracing

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	v1alpha1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

// MessagingAttributes returns OTel semantic convention attributes for messaging spans.
// busType is the EventBus or external messaging backend type (e.g., "kafka", "jetstream", "stan").
// destination is the topic/subject/channel name.
// consumerGroup is the consumer group identifier (omitted if empty).
// serverAddr is the broker/server address.
func MessagingAttributes(busType, destination, consumerGroup, serverAddr string) []attribute.KeyValue {
	system := busType
	switch busType {
	case "jetstream", "stan":
		system = "nats"
	}

	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", system),
		attribute.String("messaging.destination.name", destination),
		attribute.String("server.address", serverAddr),
	}

	if consumerGroup != "" {
		attrs = append(attrs, attribute.String("messaging.consumer.group.name", consumerGroup))
	}

	return attrs
}

// SourceTypeSpanKind maps an Argo Events EventSource type to the correct inbound
// span kind based on how it receives events from its external source.
func SourceTypeSpanKind(sourceType string) trace.SpanKind {
	switch sourceType {
	// HTTP webhook receivers -> SERVER
	case "webhook", "github", "gitlab", "bitbucket", "bitbucketserver",
		"slack", "stripe", "storagegrid", "sns", "generic":
		return trace.SpanKindServer

	// Message/event subscribers -> CONSUMER
	case "kafka", "amqp", "nats", "nsq", "mqtt", "gcppubsub",
		"redis", "redisStream", "sqs", "azureEventsHub",
		"azureQueueStorage", "azureServiceBus", "pulsar",
		"emitter", "minio":
		return trace.SpanKindConsumer

	// Pollers/watchers -> CLIENT
	case "gerrit", "sftp", "hdfs", "resource":
		return trace.SpanKindClient

	// Local/scheduled and unknown -> INTERNAL
	default:
		return trace.SpanKindInternal
	}
}

// TriggerTypeSpanKind determines the correct span kind for a sensor trigger based on
// the TriggerTemplate. It inspects which trigger field is non-nil to classify the trigger.
//
// CLIENT triggers make outbound requests to external services:
//   - HTTP, K8s, ArgoWorkflow, AWSLambda, CustomTrigger (gRPC), Slack, OpenWhisk, Email
//
// PRODUCER triggers publish messages to messaging systems:
//   - Kafka, NATS, Pulsar, AzureEventHubs, AzureServiceBus
//
// INTERNAL triggers perform local operations:
//   - Log
func TriggerTypeSpanKind(t *v1alpha1.TriggerTemplate) trace.SpanKind {
	switch {
	// Messaging producers
	case t.Kafka != nil:
		return trace.SpanKindProducer
	case t.NATS != nil:
		return trace.SpanKindProducer
	case t.Pulsar != nil:
		return trace.SpanKindProducer
	case t.AzureEventHubs != nil:
		return trace.SpanKindProducer
	case t.AzureServiceBus != nil:
		return trace.SpanKindProducer

	// Local I/O
	case t.Log != nil:
		return trace.SpanKindInternal

	// All other triggers are outbound API/HTTP/gRPC calls -> CLIENT
	default:
		return trace.SpanKindClient
	}
}
