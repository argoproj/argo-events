package alibabacloudmns

import (
	"context"
	"encoding/json"
	"fmt"

	ali_mns "github.com/aliyun/aliyun-mns-go-sdk"
	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	"github.com/argoproj/argo-events/pkg/shared/util"
)

type EventListener struct {
	EventSourceName string
	EventName       string
	MNSEventSource  aev1.MNSEventSource
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
func (el *EventListener) GetEventSourceType() aev1.EventSourceType {
	return aev1.MNSEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...common.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the ALI MNS event source...")
	defer sources.Recover(el.GetEventName())

	mnsClient, err := el.createMnsClient(ctx)
	if err != nil {
		return err
	}
	queue := ali_mns.NewMNSQueue(el.MNSEventSource.Queue, *mnsClient, 50)

	respChan := make(chan ali_mns.MessageReceiveResponse)
	errChan := make(chan error)
	received := make(chan struct{})
	for {
		// goroutine for mns client queue to block wait
		go func() {
			queue.ReceiveMessage(respChan, errChan, 30)
			received <- struct{}{}
		}()

		// goroutine to wait/handle response channel data
		go func() {
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
						return
					}
					log.Info("dispatching the event on data channel...")
					if err = dispatch(eventBytes); err != nil {
						log.Errorf("failed to dispatch ali mns event, %w", err)
						return
					}
					if e := queue.DeleteMessage(resp.ReceiptHandle); e != nil {
						log.Errorf("delete err: %v\n", e.Error())
					}
				}
			case err := <-errChan:
				{
					log.Errorf("receive err: %v\n", err)
				}
			}
		}()

		select {
		case <-ctx.Done():
			log.Info("exiting MNS event listener...")
			return nil
		case <-received:
			// next
		}
	}
}

func (el *EventListener) createMnsClient(ctx context.Context) (*ali_mns.MNSClient, error) {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	accessKey, err := util.GetSecretFromVolume(el.MNSEventSource.AccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the access key name from secret %s, %w", el.MNSEventSource.AccessKey.Name, err)
	}
	accessSecret, err := util.GetSecretFromVolume(el.MNSEventSource.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the access secret name from secret %s, %w", el.MNSEventSource.AccessKey.Name, err)
	}
	log.Infof("el.MNSEventSource.Endpoint is: %v", el.MNSEventSource.Endpoint)

	conf := ali_mns.AliMNSClientConfig{
		AccessKeyId:     accessKey,
		AccessKeySecret: accessSecret,
		EndPoint:        el.MNSEventSource.Endpoint,
	}

	client := ali_mns.NewAliMNSClientWithConfig(conf)
	return &client, nil
}
