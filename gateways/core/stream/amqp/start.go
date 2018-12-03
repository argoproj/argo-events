package amqp

import (
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	amqplib "github.com/streadway/amqp"
)

// StartConfig runs a configuration
func (ce *AMQPConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	defer func() {
		gateways.Recover()
	}()

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	a, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *a).Msg("amqp configuration")

	for {
		select {
		case _, ok :=<-config.StartChan:
			if ok {
				config.Active = true
				ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
			}

		case data, ok := <-config.DataChan:
			if ok {
				err := ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: data,
				})
				if err != nil {
					config.ErrChan <- err
				}
			}

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func getDelivery(ch *amqplib.Channel, a *amqp) (<-chan amqplib.Delivery, error) {
	err := ch.ExchangeDeclare(a.ExchangeName, a.ExchangeType, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange with name %s and type %s. err: %+v", a.ExchangeName, a.ExchangeType, err)
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %s", err)
	}

	err = ch.QueueBind(q.Name, a.RoutingKey, a.ExchangeName, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind %s exchange '%s' to queue with routingKey: %s: %s", a.ExchangeType, a.ExchangeName, a.RoutingKey, err)
	}

	delivery, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin consuming messages: %s", err)
	}
	return delivery, nil
}

func (ce *AMQPConfigExecutor) listenEvents(a *amqp, config *gateways.ConfigContext) {
	conn, err := amqplib.Dial(a.URL)
	if err != nil {
		config.ErrChan <- err
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		config.ErrChan <- err
		return
	}

	delivery, err := getDelivery(ch, a)
	if err != nil {
		config.ErrChan <- err
		return
	}

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	config.StartChan <- struct{}{}

	for {
		select {
		case msg := <-delivery:
			config.DataChan <- msg.Body
		case <-config.DoneChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.Active = false
			err = conn.Close()
			if err != nil {
				ce.GatewayConfig.Log.Error().Err(err).Str("config-name", config.Data.Src).Msg("failed to close connection")
			}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
			config.ShutdownChan <- struct{}{}
			return
		}
	}
}
