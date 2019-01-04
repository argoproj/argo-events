package amqp

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	amqplib "github.com/streadway/amqp"
)

// StartEventSource starts an event source
func (ese *AMQPEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-stream-name", *eventSource.Name).Msg("operating on event source")
	a, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(a, eventSource, dataCh, errorCh, doneCh)

	return gateways.ConsumeEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
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

func (ese *AMQPEventSourceExecutor) listenEvents(a *amqp, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	conn, err := amqplib.Dial(a.URL)
	if err != nil {
		errorCh <- err
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		errorCh <- err
		return
	}

	delivery, err := getDelivery(ch, a)
	if err != nil {
		errorCh <- err
		return
	}

	for {
		select {
		case msg := <-delivery:
			dataCh <- msg.Body
		case <-doneCh:
			err = conn.Close()
			if err != nil {
				ese.Log.Error().Err(err).Str("event-stream-name", *eventSource.Name).Msg("failed to close connection")
			}
			return
		}
	}
}
