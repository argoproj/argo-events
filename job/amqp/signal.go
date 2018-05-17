/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package amqp

import (
	"github.com/blackrock/axis/job"
	"go.uber.org/zap"

	amqplib "github.com/streadway/amqp"
)

type amqp struct {
	job.AbstractSignal
	conn     *amqplib.Connection
	delivery <-chan amqplib.Delivery
}

func (a *amqp) Start(events chan job.Event) error {
	var err error
	//todo: add support for passing in config with credentials
	a.conn, err = amqplib.Dial(a.AMQP.URL)
	if err != nil {
		a.Log.Warn("failed to connect to RabbitMQ", zap.String("url", a.AMQP.URL))
		return err
	}

	ch, err := a.conn.Channel()
	if err != nil {
		a.Log.Warn("failed to open channel on RabbitMQ")
		return err
	}

	err = ch.ExchangeDeclare(a.AMQP.ExchangeName, a.AMQP.ExchangeType, true, false, false, false, nil)
	if err != nil {
		a.Log.Warn("failed to declare RabbitMQ exchange", zap.String("name", a.AMQP.ExchangeName), zap.String("type", a.AMQP.ExchangeType))
		return err
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		a.Log.Warn("failed to declare RabbitMQ queue")
		return err
	}

	err = ch.QueueBind(q.Name, a.AMQP.RoutingKey, a.AMQP.ExchangeName, false, nil)
	if err != nil {
		a.Log.Warn("failed to bind RabbitMQ exchange to queue", zap.String("queueName", q.Name), zap.String("exchangeName", a.AMQP.ExchangeName), zap.String("key", a.AMQP.RoutingKey))
		return err
	}
	a.delivery, err = ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		a.Log.Warn("failed to begin consuming RabbitMQ messages", zap.String("queueName", q.Name))
		return err
	}

	go a.listen(events)
	return nil
}

func (a *amqp) Stop() error {
	return a.conn.Close()
}

func (a *amqp) listen(events chan job.Event) {
	for msg := range a.delivery {
		event := &event{
			amqp:     a,
			delivery: msg,
		}
		// perform constraint checks
		err := a.CheckConstraints(event.GetTimestamp())
		if err != nil {
			event.SetError(err)
		}
		a.Log.Debug("sending amqp event", zap.String("nodeID", event.GetID()))
		events <- event
	}
}
