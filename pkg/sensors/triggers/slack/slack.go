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
package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	notifications "github.com/argoproj/notifications-engine/pkg/services"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/sensors/triggers"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

type SlackTrigger struct {
	// Sensor refer to the sensor object
	Sensor *v1alpha1.Sensor
	// Trigger refers to the trigger resource
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *zap.SugaredLogger
	// http client to invoke function.
	httpClient *http.Client
	// slackSvc refers to the Slack notification service.
	slackSvc notifications.NotificationService
}

// NewSlackTrigger returns a new Slack trigger context
func NewSlackTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger, httpClient *http.Client) (*SlackTrigger, error) {
	slackTrigger := trigger.Template.Slack
	slackToken, err := sharedutil.GetSecretFromVolume(slackTrigger.SlackToken)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the slack token, %w", err)
	}

	slackSvc := notifications.NewSlackService(notifications.SlackOptions{
		Token:    slackToken,
		Username: slackTrigger.Sender.Username,
		Icon:     slackTrigger.Sender.Icon,
	})

	return &SlackTrigger{
		Sensor:     sensor,
		Trigger:    trigger,
		Logger:     logger.With(logging.LabelTriggerType, v1alpha1.TriggerTypeSlack),
		httpClient: httpClient,
		slackSvc:   slackSvc,
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *SlackTrigger) GetTriggerType() v1alpha1.TriggerType {
	return v1alpha1.TriggerTypeSlack
}

func (t *SlackTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.Slack, nil
}

func (t *SlackTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the Slack trigger resource, %w", err)
	}
	parameters := t.Trigger.Template.Slack.Parameters

	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, t.Trigger.Template.Slack.Parameters, events)
		if err != nil {
			return nil, err
		}

		var st *v1alpha1.SlackTrigger
		if err := json.Unmarshal(updatedResourceBytes, &st); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated Slack trigger resource after applying resource parameters, %w", err)
		}

		return st, nil
	}

	return resource, nil
}

// Execute executes the trigger
func (t *SlackTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	t.Logger.Info("executing SlackTrigger")
	_, ok := resource.(*v1alpha1.SlackTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to marshal the Slack trigger resource")
	}

	slackTrigger := t.Trigger.Template.Slack

	channel := slackTrigger.Channel
	if channel == "" {
		return nil, fmt.Errorf("no slack channel provided")
	}
	channel = strings.TrimPrefix(channel, "#")

	message := slackTrigger.Message
	attachments := slackTrigger.Attachments
	blocks := slackTrigger.Blocks
	if message == "" && attachments == "" && blocks == "" {
		return nil, fmt.Errorf("no text to post: At least one of message/attachments/blocks should be provided")
	}

	t.Logger.Infow("posting to channel...", zap.Any("channelName", channel))

	notification := notifications.Notification{
		Message: message,
		Slack: &notifications.SlackNotification{
			GroupingKey:     slackTrigger.Thread.MessageAggregationKey,
			NotifyBroadcast: slackTrigger.Thread.BroadcastMessageToChannel,
			Blocks:          blocks,
			Attachments:     attachments,
		},
	}
	destination := notifications.Destination{
		Service:   "slack",
		Recipient: channel,
	}
	err := t.slackSvc.Send(notification, destination)
	if err != nil {
		t.Logger.Errorw("unable to post to channel", zap.Any("channelName", channel), zap.Error(err))
		return nil, fmt.Errorf("failed to post to channel %s, %w", channel, err)
	}

	t.Logger.Infow("message successfully sent to channel", zap.Any("message", message), zap.Any("channelName", channel))
	t.Logger.Info("finished executing SlackTrigger")
	return nil, nil
}

// No Policies for SlackTrigger
func (t *SlackTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	return nil
}
