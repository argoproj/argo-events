package v1alpha1

import (
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	corev1 "k8s.io/api/core/v1"
)

// KafkaBus holds the KafkaBus EventBus information
type KafkaBus struct {
	// Exotic holds an exotic Kafka config
	Exotic *KafkaConfig `json:"exotic,omitempty" protobuf:"bytes,1,opt,name=exotic"`
}

type KafkaConfig struct {
	// URL to kafka cluster, multiple URLs separated by comma
	URL string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	// Kafka version, sarama defaults to the oldest supported stable version
	// +optional
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	// Topic name, defaults to namespace_name.eventbus_name
	// +optional
	Topic string `json:"topic,omitempty" protobuf:"bytes,3,opt,name=topic"`
	// TLS configuration for the kafka client.
	// +optional
	TLS *apicommon.TLSConfig `json:"tls,omitempty" protobuf:"bytes,4,opt,name=tls"`
	// SASL configuration for the kafka client
	// +optional
	SASL *apicommon.SASLConfig `json:"sasl,omitempty" protobuf:"bytes,5,opt,name=sasl"`
	// Consumer group for kafka client
	// +optional
	ConsumerGroup *KafkaConsumerGroup `json:"consumerGroup,omitempty" protobuf:"bytes,6,opt,name=consumerGroup"`
	// Secret for auth
	// +optional
	AccessSecret *corev1.SecretKeySelector `json:"accessSecret,omitempty" protobuf:"bytes,7,opt,name=accessSecret"`
}

type KafkaConsumerGroup struct {
	// The name for the consumer group to use
	GroupName string `json:"groupName,omitempty" protobuf:"bytes,1,opt,name=groupName"`
	// Rebalance strategy can be one of: sticky, roundrobin, range. Range is the default.
	// +optional
	RebalanceStrategy string `json:"rebalanceStrategy,omitempty" protobuf:"bytes,2,opt,name=rebalanceStrategy"`
	// When starting up a new group do we want to start from the oldest event (true) or the newest event (false), defaults to false
	// +optional
	StartOldest bool `json:"startOldest,omitempty" default:"false" protobuf:"bytes,3,opt,name=startOldest"`
}
