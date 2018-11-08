package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
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

// Runs a configuration
func (ce *KafkaConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	k, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *k).Msg("kafka configuration")

	for {
		select {
		case <-config.StartChan:
			config.Active = true
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running")

		case data := <-config.DataChan:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *KafkaConfigExecutor) listenEvents(k *kafka, config *gateways.ConfigContext) {
	consumer, err := sarama.NewConsumer([]string{k.URL}, nil)
	if err != nil {
		config.ErrChan <- err
		return
	}

	pInt, err := strconv.ParseInt(k.Partition, 10, 32)
	if err != nil {
		config.ErrChan <- err
		return
	}
	partition := int32(pInt)

	availablePartitions, err := consumer.Partitions(k.Topic)
	if err != nil {
		config.ErrChan <- err
		return
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		config.ErrChan <- err
		return
	}

	partitionConsumer, err := consumer.ConsumePartition(k.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		config.ErrChan <- err
		return
	}

	config.StartChan <- struct{}{}

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	for {
		select {
		case msg := <-partitionConsumer.Messages():
			config.DataChan <- msg.Value

		case err := <-partitionConsumer.Errors():
			ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Str("partition", k.Partition).Str("topic", k.Topic).Err(err).Msg("received an error")
			config.ErrChan <- err
			return

		case <-config.DoneChan:
			err = partitionConsumer.Close()
			if err != nil {
				ce.GatewayConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to close consumer")
			}
			config.ShutdownChan <- struct{}{}
			return
		}
	}
}
