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

package alibabacloudmns

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	ali_mns "github.com/aliyun/aliyun-mns-go-sdk"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for aws sqs event source
type EventListener struct {
	EventSourceName string
	EventName       string
	MNSEventSource  v1alpha1.MNSEventSource
	Metrics         *metrics.Metrics
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.MNSEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the ALI MNS event source...")
	defer sources.Recover(el.GetEventName())
	mnsClient, err := el.createMnsClient(ctx)
	if err != nil {
		return err
	}
	queue := ali_mns.NewMNSQueue(el.MNSEventSource.Queue, *mnsClient, 50)
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM)
	respChan := make(chan ali_mns.MessageReceiveResponse)
	errChan := make(chan error)
	endChan := make(chan int)
	for {
		select {
		case <-ctx.Done():
			log.Info("exiting MNS event listener...")
			return nil
		default:
		}
		go func() {
			sig := <-signalChannel
			switch sig {
			case syscall.SIGTERM:
				log.Info("receieved syscall.SIGTERM...")
				endChan <- 1
				return
			default:
			}
			select {
			case resp := <-respChan:
				{
					log.Infof("response: %v \n", resp)
					eventData := &events.MNSEventData{
						MessageId: resp.MessageId,
						Body:      resp.MessageBody,
					}
					eventBytes, err := json.Marshal(eventData)
					if err != nil {
						log.Errorf("failed to marshal the event data, rejecting the event, %w", err)
						endChan <- 1
						return
					}
					log.Info("dispatching the event on data channel...")
					if err = dispatch(eventBytes); err != nil {
						log.Errorf("failed to dispatch ali mns event, %w", err)
						endChan <- 1
						return
					}
					if e := queue.DeleteMessage(resp.ReceiptHandle); e != nil {
						log.Errorf("delete err: %v\n", e.Error())
					}
					endChan <- 1
				}
			case err := <-errChan:
				{
					log.Errorf("receive err: %v\n", err)
					endChan <- 1
				}
			}
		}()
		queue.ReceiveMessage(respChan, errChan, 30)
		<-endChan
	}
}

func (el *EventListener) createMnsClient(ctx context.Context) (*ali_mns.MNSClient, error) {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	accessKey, err := common.GetSecretFromVolume(el.MNSEventSource.AccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the access key name from secret %s, %w", el.MNSEventSource.AccessKey.Name, err)
	}
	accessSecret, err := common.GetSecretFromVolume(el.MNSEventSource.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the access secret name from secret %s, %w", el.MNSEventSource.AccessKey.Name, err)
	}
	log.Infof("el.MNSEventSource.Endpoint is: %v", el.MNSEventSource.Endpoint)
	client := ali_mns.NewAliMNSClient(el.MNSEventSource.Endpoint,
		accessKey,
		accessSecret)
	return &client, nil
}
