package kafka

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/Shopify/sarama"
	"strconv"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/argoproj/argo-events/common"
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
func (ce *KafkaConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer ce.GatewayConfig.GatewayCleanup(config, &errMessage, err)

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	k, err := parseConfig(config.Data.Config)
	if err != nil {
		return err
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
		}
	}
	
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

func (ce *KafkaConfigExecutor) listenEvents(k *kafka, config *gateways.ConfigContext) {

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	k, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- err
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *k).Msg("kafka configuration")

	consumer, err := sarama.NewConsumer([]string{k.URL}, nil)
	if err != nil {
		config.ErrChan <- err
	}

	pInt, err := strconv.ParseInt(k.Partition, 10, 32)
	if err != nil {
		config.ErrChan <- err
	}
	partition := int32(pInt)

	availablePartitions, err := consumer.Partitions(k.Topic)
	if err != nil {
		config.ErrChan <- err
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		config.ErrChan <- err
	}

	partitionConsumer, err := consumer.ConsumePartition(k.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		config.ErrChan <- err
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

		case <-config.StopChan:
			config.Active = false
			err = partitionConsumer.Close()
			if err != nil {
				config.ErrChan <- err
				return
			}
			config.DoneChan <- struct{}{}
		}
	}
}
