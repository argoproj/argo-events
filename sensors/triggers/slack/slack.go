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
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
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
}

// NewSlackTrigger returns a new Slack trigger context
func NewSlackTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger, httpClient *http.Client) (*SlackTrigger, error) {
	return &SlackTrigger{
		Sensor:     sensor,
		Trigger:    trigger,
		Logger:     logger.With(logging.LabelTriggerType, apicommon.SlackTrigger),
		httpClient: httpClient,
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *SlackTrigger) GetTriggerType() apicommon.TriggerType {
	return apicommon.SlackTrigger
}

func (t *SlackTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.Slack, nil
}

func (t *SlackTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the Slack trigger resource")
	}
	parameters := t.Trigger.Template.Slack.Parameters

	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, t.Trigger.Template.Slack.Parameters, events)
		if err != nil {
			return nil, err
		}

		var st *v1alpha1.SlackTrigger
		if err := json.Unmarshal(updatedResourceBytes, &st); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the updated Slack trigger resource after applying resource parameters")
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
		return nil, errors.New("failed to marshal the Slack trigger resource")
	}

	slacktrigger := t.Trigger.Template.Slack

	channel := slacktrigger.Channel
	if channel == "" {
		return nil, errors.New("no slack channel provided")
	}
	channel = strings.TrimPrefix(channel, "#")

	message := slacktrigger.Message
	if message == "" {
		return nil, errors.New("no slack message to post")
	}

	slackToken, err := common.GetSecretFromVolume(slacktrigger.SlackToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve the slack token")
	}

	api := slack.New(slackToken, slack.OptionDebug(false))

	t.Logger.Infow("posting to channel...", zap.Any("channelName", channel))
	for {
		channelID, timestamp, err := api.PostMessage(channel, slack.MsgOptionText(message, false))
		if err != nil {
			if err.Error() == "not_in_channel" {
				isPrivateChannel := false
				params := &slack.GetConversationsParameters{
					Limit:           200,
					Types:           []string{"public_channel", "private_channel"},
					ExcludeArchived: "true",
				}

				for {
					channels, nextCursor, err := api.GetConversations(params)
					if err != nil {
						switch e := err.(type) {
						case *slack.RateLimitedError:
							<-time.After(e.RetryAfter)
							continue
						default:
							t.Logger.Errorw("unable to list channels", zap.Error(err))
							return nil, errors.Wrapf(err, "failed to list channels")
						}
					}
					for _, c := range channels {
						if c.Name == channel {
							channelID = c.ID
							isPrivateChannel = c.IsPrivate
							break
						}
					}
					if nextCursor == "" || channelID != "" {
						break
					}
					params.Cursor = nextCursor
				}
				if channelID == "" {
					return nil, errors.Errorf("failed to get channelID of %s", channel)
				}
				if isPrivateChannel {
					return nil, errors.Errorf("cannot join private channel %s", channel)
				}

				c, _, _, err := api.JoinConversation(channelID)
				if err != nil {
					t.Logger.Errorw("unable to join channel...", zap.Any("channelName", channel), zap.Any("channelID", channelID), zap.Error(err))
					return nil, errors.Wrapf(err, "failed to join channel %s", channel)
				}
				t.Logger.Debugw("successfully joined channel", zap.Any("channel", c))
				continue
			} else {
				t.Logger.Errorw("unable to post to channel...", zap.Any("channelName", channel), zap.Error(err))
				return nil, errors.Wrapf(err, "failed to post to channel %s", channel)
			}
		}
		t.Logger.Infow("message successfully sent to channelID with timestamp", zap.Any("message", message), zap.Any("channelID", channelID), zap.Any("timestamp", timestamp))
		t.Logger.Info("finished executing SlackTrigger")
		return nil, nil
	}
}

// No Policies for SlackTrigger
func (t *SlackTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	return nil
}
