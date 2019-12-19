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

package sensors

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-events/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/nats-io/go-nats"
	snats "github.com/nats-io/go-nats-streaming"
)

func (sec *sensorExecutionCtx) successNatsConnection() {
	labels := map[string]string{
		common.LabelEventType:  string(common.OperationSuccessEventType),
		common.LabelSensorName: sec.sensor.Name,
		common.LabelOperation:  "nats_connection_setup",
	}
	if err := common.GenerateK8sEvent(sec.kubeClient, "connection setup successfully", common.OperationSuccessEventType, "connection setup", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
		sec.log.WithError(err).Error("failed to create K8s event to log nats connection setup success")
		return
	}
	sec.log.Info("created event for nats connection setup success")
}

func (sec *sensorExecutionCtx) escalateNatsConnectionFailure() {
	// escalate error
	labels := map[string]string{
		common.LabelEventType:  string(common.OperationFailureEventType),
		common.LabelSensorName: sec.sensor.Name,
		common.LabelOperation:  "nats_connection_setup",
	}
	if err := common.GenerateK8sEvent(sec.kubeClient, "connection setup failed", common.OperationFailureEventType, "connection setup", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
		sec.log.WithError(err).Error("failed to create K8s event to log nats connection setup error")
		return
	}
	sec.log.Warn("created event for nats connection failure")
}

func (sec *sensorExecutionCtx) successNatsSubscription(eventSource string) {
	labels := map[string]string{
		common.LabelEventType:   string(common.OperationSuccessEventType),
		common.LabelSensorName:  sec.sensor.Name,
		common.LabelEventSource: strings.Replace(eventSource, ":", "_", -1),
		common.LabelOperation:   "nats_subscription_success",
	}
	if err := common.GenerateK8sEvent(sec.kubeClient, "nats subscription success", common.OperationSuccessEventType, "subscription setup", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
		sec.log.WithField(common.LabelEventSource, eventSource).WithError(err).Error("failed to create K8s event to log nats subscription success")
		return
	}
	sec.log.WithField(common.LabelEventSource, eventSource).Info("created event for nats subscription success ")
}

func (sec *sensorExecutionCtx) escalateNatsSubscriptionFailure(eventSource string) {
	// escalate error
	labels := map[string]string{
		common.LabelEventType:   string(common.OperationFailureEventType),
		common.LabelSensorName:  sec.sensor.Name,
		common.LabelEventSource: strings.Replace(eventSource, ":", "_", -1),
		common.LabelOperation:   "nats_subscription_failure",
	}
	if err := common.GenerateK8sEvent(sec.kubeClient, "nats subscription failed", common.OperationFailureEventType, "subscription setup", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
		sec.log.WithField(common.LabelEventSource, eventSource).Error("failed to create K8s event to log nats subscription error")
		return
	}
	sec.log.WithField(common.LabelEventSource, eventSource).Warn("created event for nats subscription failure")
}

// NatsEventProtocol handles events sent over NATS
// Queue subscription is used because lets say sensor is running with couple of dependencies and the subscription is
// established for them and then sensor resource failed to persist updates. So for the next sensor resource update, the dependency connected
// status will evaluate as false which will cause another subscription for that dependency. This will result in two subscriptions for the same dependency
// trying to put a message twice on internal queue.
func (sec *sensorExecutionCtx) NatsEventProtocol() {
	var err error

	switch sec.sensor.Spec.EventProtocol.Nats.Type {
	case pc.Standard:
		if sec.nconn.standard == nil {
			sec.nconn.standard, err = nats.Connect(sec.sensor.Spec.EventProtocol.Nats.URL)
			if err != nil {
				// escalate failure
				sec.escalateNatsConnectionFailure()
				sec.log.WithError(err).Panic("failed to connect to nats server")
			}
			// log success
			sec.successNatsConnection()
		}
		for _, dependency := range sec.sensor.Spec.Dependencies {
			if dependency.Connected {
				continue
			}
			if _, err := sec.getNatsStandardSubscription(dependency.Name); err != nil {
				// escalate failure
				sec.escalateNatsSubscriptionFailure(dependency.Name)
				sec.log.WithField(common.LabelEventSource, dependency.Name).Error("failed to get the nats subscription")
				continue
			}
			dependency.Connected = true
			// log success
			sec.successNatsSubscription(dependency.Name)
		}

	case pc.Streaming:
		if sec.nconn.stream == nil {
			sec.nconn.stream, err = snats.Connect(sec.sensor.Spec.EventProtocol.Nats.ClusterId, sec.sensor.Spec.EventProtocol.Nats.ClientId, snats.NatsURL(sec.sensor.Spec.EventProtocol.Nats.URL))
			if err != nil {
				sec.escalateNatsConnectionFailure()
				sec.log.WithError(err).Panic("failed to connect to nats streaming server")
			}
			sec.successNatsConnection()
		}
		for _, dependency := range sec.sensor.Spec.Dependencies {
			if dependency.Connected {
				continue
			}
			if _, err := sec.getNatsStreamingSubscription(dependency.Name); err != nil {
				sec.escalateNatsSubscriptionFailure(dependency.Name)
				sec.log.WithField(common.LabelEventSource, dependency.Name).WithError(err).Error("failed to get the nats subscription")
				continue
			}
			dependency.Connected = true
			sec.successNatsSubscription(dependency.Name)
		}
	}
}

// getNatsStandardSubscription returns a standard nats subscription
func (sec *sensorExecutionCtx) getNatsStandardSubscription(eventSource string) (*nats.Subscription, error) {
	return sec.nconn.standard.QueueSubscribe(eventSource, common.DefaultNatsQueueName(sec.sensor.Name, eventSource), func(msg *nats.Msg) {
		sec.processNatsMessage(msg.Data, eventSource)
	})
}

// getNatsStreamingSubscription returns a streaming nats subscription
func (sec *sensorExecutionCtx) getNatsStreamingSubscription(eventSource string) (snats.Subscription, error) {
	subscriptionOption, err := sec.getNatsStreamingOption(eventSource)
	if err != nil {
		return nil, err
	}
	return sec.nconn.stream.QueueSubscribe(eventSource, common.DefaultNatsQueueName(sec.sensor.Name, eventSource), func(msg *snats.Msg) {
		sec.processNatsMessage(msg.Data, eventSource)
	}, subscriptionOption)
}

// getNatsStreamingOption returns a streaming option
func (sec *sensorExecutionCtx) getNatsStreamingOption(eventSource string) (snats.SubscriptionOption, error) {
	if sec.sensor.Spec.EventProtocol.Nats.StartWithLastReceived {
		snats.StartWithLastReceived()
	}
	if sec.sensor.Spec.EventProtocol.Nats.DeliverAllAvailable {
		snats.DeliverAllAvailable()
	}
	if sec.sensor.Spec.EventProtocol.Nats.Durable {
		snats.DurableName(eventSource)
	}
	if sec.sensor.Spec.EventProtocol.Nats.StartAtSequence != "" {
		sequence, err := strconv.Atoi(sec.sensor.Spec.EventProtocol.Nats.StartAtSequence)
		if err != nil {
			return nil, err
		}
		return snats.StartAtSequence(uint64(sequence)), nil
	}
	if sec.sensor.Spec.EventProtocol.Nats.StartAtTime != "" {
		startTime, err := time.Parse(common.StandardTimeFormat, sec.sensor.Spec.EventProtocol.Nats.StartAtTime)
		if err != nil {
			return nil, err
		}
		return snats.StartAtTime(startTime), nil
	}
	if sec.sensor.Spec.EventProtocol.Nats.StartAtTimeDelta != "" {
		duration, err := time.ParseDuration(sec.sensor.Spec.EventProtocol.Nats.StartAtTimeDelta)
		if err != nil {
			return nil, err
		}
		return snats.StartAtTimeDelta(duration), nil
	}
	return func(o *snats.SubscriptionOptions) error {
		return nil
	}, nil
}

// processNatsMessage handles a nats message payload
func (sec *sensorExecutionCtx) processNatsMessage(msg []byte, eventSource string) {
	log := sec.log.WithField(common.LabelEventSource, eventSource)
	event, err := sec.parseEvent(msg)
	if err != nil {
		log.WithError(err).Error("failed to parse message into event")
		return
	}
	// validate whether the event is from gateway that this sensor is watching and send event over internal queue if valid
	if sec.sendEventToInternalQueue(event, nil) {
		sec.log.Info("event successfully sent over internal queue")
		return
	}
	sec.log.Warn("event is from unknown source")
}
