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
	"github.com/argoproj/argo-events/pkg/apis/events"
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

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")
	channels := server.NewChannels()

	go server.HandleEventsFromEventSource(eventSource.Name, eventStream, channels, listener.Logger)

	defer func() {
		channels.Stop <- struct{}{}
	}()

	if err := listener.listenEvents(eventSource, channels); err != nil {
		listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to listen to events")
		return err
	}

	return nil
}

// listenEvents listens events published by redis
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var redisEventSource *v1alpha1.RedisEventSource
	if err := yaml.Unmarshal(eventSource.Value, &redisEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
	}

	logger.Infoln("retrieving password if it has been configured...")
	password, err := listener.getPassword(redisEventSource)
	if err != nil {
		return err
	}

	logger.Infoln("setting up a redis client...")
	client := redis.NewClient(&redis.Options{
		Addr:     redisEventSource.HostAddress,
		Password: password,
		DB:       redisEventSource.DB,
	})

	if status := client.Ping(); status.Err() != nil {
		return errors.Wrapf(status.Err(), "failed to connect to host %s and db %d for event source %s", redisEventSource.HostAddress, redisEventSource.DB, eventSource.Name)
	}

	pubsub := client.Subscribe(redisEventSource.Channels...)
	// Wait for confirmation that subscription is created before publishing anything.
	if _, err := pubsub.Receive(); err != nil {
		return errors.Wrapf(err, "failed to receive the subscription confirmation for event source %s", eventSource.Name)
	}

	// Go channel which receives messages.
	ch := pubsub.Channel()

	for {
		select {
		case message := <-ch:
			logger.WithField("channel", message.Channel).Infoln("received a message")
			eventData := &events.RedisEventData{
				Channel: message.Channel,
				Pattern: message.Pattern,
				Body:    message.Payload,
			}
			eventBody, err := json.Marshal(&eventData)
			if err != nil {
				logger.WithError(err).WithField("channel", message.Channel).Errorln("failed to marshal the event data, rejecting the event...")
				continue
			}
			logger.WithField("channel", message.Channel).Infoln("dispatching th event on the data channel...")
			channels.Data <- eventBody

		case <-channels.Done:
			logger.Infoln("event source is stopped. unsubscribing the subscription")
			if err := pubsub.Unsubscribe(redisEventSource.Channels...); err != nil {
				logger.WithError(err).Errorln("failed to unsubscribe")
			}
			return nil
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
