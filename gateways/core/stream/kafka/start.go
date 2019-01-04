package kafka

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/gateways"
	"strconv"
)

func verifyPartitionAvailable(part int32, partitions []int32) bool {
	for _, p := range partitions {
		if part == p {
			return true
		}
	}
	return false
}

// StartEventSource starts an event source
func (ese *KafkaEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", *eventSource.Name).Msg("operating on event source")
	k, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(k, eventSource, dataCh, errorCh, doneCh)

	return gateways.ConsumeEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
}

func (ese *KafkaEventSourceExecutor) listenEvents(k *kafka, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	consumer, err := sarama.NewConsumer([]string{k.URL}, nil)
	if err != nil {
		errorCh <- err
		return
	}

	pInt, err := strconv.ParseInt(k.Partition, 10, 32)
	if err != nil {
		errorCh <- err
		return
	}
	partition := int32(pInt)

	availablePartitions, err := consumer.Partitions(k.Topic)
	if err != nil {
		errorCh <- err
		return
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		errorCh <- fmt.Errorf("partition %d is not available", partition)
		return
	}

	partitionConsumer, err := consumer.ConsumePartition(k.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		errorCh <- err
		return
	}

	for {
		select {
		case msg := <-partitionConsumer.Messages():
			dataCh <- msg.Value

		case err := <-partitionConsumer.Errors():
			errorCh <- err
			return

		case <-doneCh:
			err = partitionConsumer.Close()
			if err != nil {
				ese.Log.Error().Err(err).Str("event-source-name", *eventSource.Name).Msg("failed to close consumer")
			}
			return
		}
	}
}
