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
	"time"

	"github.com/blackrock/axis/job"
	amqplib "github.com/streadway/amqp"
)

type event struct {
	job.AbstractEvent
	amqp     *amqp
	delivery amqplib.Delivery
}

func (e *event) GetID() string {
	return e.amqp.AbstractSignal.GetID()
}

// GetBody returns the nats message data
func (e *event) GetBody() []byte {
	return e.delivery.Body
}

func (e *event) GetSource() string {
	return e.delivery.RoutingKey
}

func (e *event) GetSignal() job.Signal {
	return e.amqp
}

func (e *event) GetTimestamp() time.Time {
	return e.delivery.Timestamp
}

func (e *event) GetContentType() string {
	return e.delivery.ContentType
}

func (e *event) GetHeader(string) interface{} {
	return e.delivery.Headers
}
