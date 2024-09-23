package v1alpha1

// KafkaBus holds the KafkaBus EventBus information
type KafkaBus struct {
	// URL to kafka cluster, multiple URLs separated by comma
	URL string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	// Topic name, defaults to {namespace_name}-{eventbus_name}
	// +optional
	Topic string `json:"topic,omitempty" protobuf:"bytes,2,opt,name=topic"`
	// Kafka version, sarama defaults to the oldest supported stable version
	// +optional
	Version string `json:"version,omitempty" protobuf:"bytes,3,opt,name=version"`
	// TLS configuration for the kafka client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,4,opt,name=tls"`
	// SASL configuration for the kafka client
	// +optional
	SASL *SASLConfig `json:"sasl,omitempty" protobuf:"bytes,5,opt,name=sasl"`
	// Consumer group for kafka client
	// +optional
	ConsumerGroup *KafkaConsumerGroup `json:"consumerGroup,omitempty" protobuf:"bytes,6,opt,name=consumerGroup"`
}
