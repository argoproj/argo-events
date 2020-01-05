/*
Copyright 2020 BlackRock, Inc.

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

package redis

import (
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for the Redis event source
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
	// K8sClient is the kubernetes client
	K8sClient kubernetes.Interface
}

// RedisMessage holds the message received from Redis PubSub.
type RedisMessage struct {
	Channel string `json:"channel"`
	Pattern string `json:"pattern"`
	Payload string `json:"payload"`
}

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)

	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents listens events published by redis
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer server.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var redisEventSource *v1alpha1.RedisEventSource
	if err := yaml.Unmarshal(eventSource.Value, &redisEventSource); err != nil {
		errorCh <- errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
		return
	}

	password, err := listener.getPassword(redisEventSource)
	if err != nil {
		errorCh <- err
		return
	}

	logger.Infoln("setting up a redis client")

	client := redis.NewClient(&redis.Options{
		Addr:     redisEventSource.HostAddress,
		Password: password,
		DB:       redisEventSource.DB,
	})

	if status := client.Ping(); status.Err() != nil {
		errorCh <- errors.Wrapf(status.Err(), "failed to connect to host %s and db %d", redisEventSource.HostAddress, redisEventSource.DB)
		return
	}

	pubsub := client.Subscribe(redisEventSource.Channels...)
	// Wait for confirmation that subscription is created before publishing anything.
	if _, err := pubsub.Receive(); err != nil {
		errorCh <- errors.Wrapf(err, "failed to receive the subscription confirmation for event source %s", eventSource.Name)
		return
	}

	// Go channel which receives messages.
	ch := pubsub.Channel()

	for {
		select {
		case message := <-ch:
			logger.WithField("channel", message.Channel).Infoln("received a message")
			data := RedisMessage{
				Channel: message.Channel,
				Pattern: message.Pattern,
				Payload: message.Payload,
			}
			body, err := json.Marshal(&data)
			if err != nil {
				logger.WithError(err).WithField("channel", message.Channel).Errorln("failed to marshal the message")
				continue
			}
			logger.WithField("channel", message.Channel).Infoln("dispatching message on the data channel")
			dataCh <- body

		case <-doneCh:
			logger.Infoln("event source is stopped. Unsubscribe the subscription")
			if err := pubsub.Unsubscribe(redisEventSource.Channels...); err != nil {
				logger.WithError(err).Errorln("failed to unsubscribe")
				return
			}
		}
	}
}

// getPassword returns the password for authentication
func (listener *EventListener) getPassword(redisEventSource *v1alpha1.RedisEventSource) (string, error) {
	if redisEventSource.Password != nil {
		password, err := common.GetSecrets(listener.K8sClient, redisEventSource.Namespace, redisEventSource.Password)
		if err != nil {
			return "", errors.Wrapf(err, "failed to retrieve the password from secret %s within namespace %s", redisEventSource.Password.Name, redisEventSource.Namespace)
		}
		return password, nil
	}
	return "", nil
}
