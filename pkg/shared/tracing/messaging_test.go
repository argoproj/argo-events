package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"

	v1alpha1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestMessagingAttributes(t *testing.T) {
	tests := []struct {
		name          string
		busType       string
		destination   string
		consumerGroup string
		serverAddr    string
		wantSystem    string
		wantGroupAttr bool
	}{
		{
			name:          "kafka preserves system name",
			busType:       "kafka",
			destination:   "my-topic",
			consumerGroup: "my-group",
			serverAddr:    "kafka:9092",
			wantSystem:    "kafka",
			wantGroupAttr: true,
		},
		{
			name:          "jetstream maps to nats",
			busType:       "jetstream",
			destination:   "my-subject",
			consumerGroup: "my-consumer",
			serverAddr:    "nats:4222",
			wantSystem:    "nats",
			wantGroupAttr: true,
		},
		{
			name:          "stan maps to nats",
			busType:       "stan",
			destination:   "my-channel",
			consumerGroup: "my-queue",
			serverAddr:    "nats:4222",
			wantSystem:    "nats",
			wantGroupAttr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := MessagingAttributes(tt.busType, tt.destination, tt.consumerGroup, tt.serverAddr)

			attrMap := make(map[string]string)
			for _, a := range attrs {
				attrMap[string(a.Key)] = a.Value.AsString()
			}

			assert.Equal(t, tt.wantSystem, attrMap["messaging.system"])
			assert.Equal(t, tt.destination, attrMap["messaging.destination.name"])
			assert.Equal(t, tt.serverAddr, attrMap["server.address"])

			if tt.wantGroupAttr {
				assert.Equal(t, tt.consumerGroup, attrMap["messaging.consumer.group.name"])
			} else {
				_, exists := attrMap["messaging.consumer.group.name"]
				assert.False(t, exists)
			}
		})
	}
}

func TestMessagingAttributes_EmptyConsumerGroup(t *testing.T) {
	attrs := MessagingAttributes("kafka", "my-topic", "", "kafka:9092")

	for _, a := range attrs {
		assert.NotEqual(t, "messaging.consumer.group.name", string(a.Key),
			"messaging.consumer.group.name should be omitted when consumerGroup is empty")
	}

	// Verify the three mandatory attributes are still present
	assert.Len(t, attrs, 3)
}

func TestSourceTypeSpanKind(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		want       trace.SpanKind
	}{
		// HTTP webhook receivers -> SERVER
		{name: "webhook", sourceType: "webhook", want: trace.SpanKindServer},
		{name: "github", sourceType: "github", want: trace.SpanKindServer},
		{name: "gitlab", sourceType: "gitlab", want: trace.SpanKindServer},
		{name: "bitbucket", sourceType: "bitbucket", want: trace.SpanKindServer},
		{name: "bitbucketserver", sourceType: "bitbucketserver", want: trace.SpanKindServer},
		{name: "slack", sourceType: "slack", want: trace.SpanKindServer},
		{name: "stripe", sourceType: "stripe", want: trace.SpanKindServer},
		{name: "storagegrid", sourceType: "storagegrid", want: trace.SpanKindServer},
		{name: "sns", sourceType: "sns", want: trace.SpanKindServer},
		{name: "generic", sourceType: "generic", want: trace.SpanKindServer},

		// Message/event subscribers -> CONSUMER
		{name: "kafka", sourceType: "kafka", want: trace.SpanKindConsumer},
		{name: "amqp", sourceType: "amqp", want: trace.SpanKindConsumer},
		{name: "nats", sourceType: "nats", want: trace.SpanKindConsumer},
		{name: "nsq", sourceType: "nsq", want: trace.SpanKindConsumer},
		{name: "mqtt", sourceType: "mqtt", want: trace.SpanKindConsumer},
		{name: "gcppubsub", sourceType: "gcppubsub", want: trace.SpanKindConsumer},
		{name: "redis", sourceType: "redis", want: trace.SpanKindConsumer},
		{name: "redisStream", sourceType: "redisStream", want: trace.SpanKindConsumer},
		{name: "sqs", sourceType: "sqs", want: trace.SpanKindConsumer},
		{name: "azureEventsHub", sourceType: "azureEventsHub", want: trace.SpanKindConsumer},
		{name: "azureQueueStorage", sourceType: "azureQueueStorage", want: trace.SpanKindConsumer},
		{name: "azureServiceBus", sourceType: "azureServiceBus", want: trace.SpanKindConsumer},
		{name: "pulsar", sourceType: "pulsar", want: trace.SpanKindConsumer},
		{name: "emitter", sourceType: "emitter", want: trace.SpanKindConsumer},
		{name: "minio", sourceType: "minio", want: trace.SpanKindConsumer},

		// Pollers/watchers -> CLIENT
		{name: "gerrit", sourceType: "gerrit", want: trace.SpanKindClient},
		{name: "sftp", sourceType: "sftp", want: trace.SpanKindClient},
		{name: "hdfs", sourceType: "hdfs", want: trace.SpanKindClient},
		{name: "resource", sourceType: "resource", want: trace.SpanKindClient},

		// Local/scheduled -> INTERNAL
		{name: "calendar", sourceType: "calendar", want: trace.SpanKindInternal},
		{name: "file", sourceType: "file", want: trace.SpanKindInternal},

		// Unknown -> INTERNAL
		{name: "unknown_source", sourceType: "unknown_source", want: trace.SpanKindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SourceTypeSpanKind(tt.sourceType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTriggerTypeSpanKind(t *testing.T) {
	tests := []struct {
		name     string
		template *v1alpha1.TriggerTemplate
		want     trace.SpanKind
	}{
		// CLIENT triggers (outbound API calls)
		{"HTTP", &v1alpha1.TriggerTemplate{HTTP: &v1alpha1.HTTPTrigger{}}, trace.SpanKindClient},
		{"K8s", &v1alpha1.TriggerTemplate{K8s: &v1alpha1.StandardK8STrigger{}}, trace.SpanKindClient},
		{"ArgoWorkflow", &v1alpha1.TriggerTemplate{ArgoWorkflow: &v1alpha1.ArgoWorkflowTrigger{}}, trace.SpanKindClient},
		{"AWSLambda", &v1alpha1.TriggerTemplate{AWSLambda: &v1alpha1.AWSLambdaTrigger{}}, trace.SpanKindClient},
		{"CustomTrigger", &v1alpha1.TriggerTemplate{CustomTrigger: &v1alpha1.CustomTrigger{}}, trace.SpanKindClient},
		{"Slack", &v1alpha1.TriggerTemplate{Slack: &v1alpha1.SlackTrigger{}}, trace.SpanKindClient},
		{"OpenWhisk", &v1alpha1.TriggerTemplate{OpenWhisk: &v1alpha1.OpenWhiskTrigger{}}, trace.SpanKindClient},
		{"Email", &v1alpha1.TriggerTemplate{Email: &v1alpha1.EmailTrigger{}}, trace.SpanKindClient},

		// PRODUCER triggers (messaging)
		{"Kafka", &v1alpha1.TriggerTemplate{Kafka: &v1alpha1.KafkaTrigger{}}, trace.SpanKindProducer},
		{"NATS", &v1alpha1.TriggerTemplate{NATS: &v1alpha1.NATSTrigger{}}, trace.SpanKindProducer},
		{"Pulsar", &v1alpha1.TriggerTemplate{Pulsar: &v1alpha1.PulsarTrigger{}}, trace.SpanKindProducer},
		{"AzureEventHubs", &v1alpha1.TriggerTemplate{AzureEventHubs: &v1alpha1.AzureEventHubsTrigger{}}, trace.SpanKindProducer},
		{"AzureServiceBus", &v1alpha1.TriggerTemplate{AzureServiceBus: &v1alpha1.AzureServiceBusTrigger{}}, trace.SpanKindProducer},

		// INTERNAL triggers (local)
		{"Log", &v1alpha1.TriggerTemplate{Log: &v1alpha1.LogTrigger{}}, trace.SpanKindInternal},

		// Empty template defaults to CLIENT
		{"empty", &v1alpha1.TriggerTemplate{}, trace.SpanKindClient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TriggerTypeSpanKind(tt.template)
			assert.Equal(t, tt.want, got)
		})
	}
}
