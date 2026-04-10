package tracing

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
