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
	"fmt"

	"github.com/argoproj/argo-events/job"
	amqplib "github.com/streadway/amqp"
	"go.uber.org/zap"
)

const (
	// StreamTypeAMQP defines the exact string representation for AMQP stream types
	StreamTypeAMQP  = "AMQP"
	exchangeNameKey = "exchangeName"
	exchangeTypeKey = "exchangeType"
	routingKeyKey   = "routingKey"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) (job.Signal, error) {
	abstract.Log.Info("creating signal", zap.String("type", abstract.Stream.Type))
	amqp := &amqp{
		AbstractSignal: abstract,
		delivery:       make(<-chan amqplib.Delivery),
	}
	// parse attributes
	var ok bool
	if amqp.exchangeName, ok = abstract.Stream.Attributes[exchangeNameKey]; !ok {
		return nil, fmt.Errorf(job.ErrMissingAttribute, abstract.Stream.Type, exchangeNameKey)
	}
	if amqp.exchangeType, ok = abstract.Stream.Attributes[exchangeTypeKey]; !ok {
		return nil, fmt.Errorf(job.ErrMissingAttribute, abstract.Stream.Type, exchangeTypeKey)
	}
	if amqp.routingKey, ok = abstract.Stream.Attributes[routingKeyKey]; !ok {
		return nil, fmt.Errorf(job.ErrMissingAttribute, abstract.Stream.Type, routingKeyKey)
	}

	return amqp, nil
}

// AMQP will be added to the executor session
func AMQP(es *job.ExecutorSession) {
	es.AddStreamFactory(StreamTypeAMQP, &factory{})
}
