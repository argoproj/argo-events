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
	//"bytes"
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	"k8s.io/client-go/kubernetes"

	//"github.com/argoproj/argo-events/common"
	"net/http"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	//offical:
	//"github.com/slack-go/slack"

	//old:
	"github.com/nlopes/slack"
)

type SlackTrigger struct {
	// K8sClient is the Kubernetes client
	K8sClient kubernetes.Interface
	// Sensor refer to the sensor object
	Sensor *v1alpha1.Sensor
	// Trigger refers to the trigger resource
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *logrus.Logger
	// http client to invoke function.
	httpClient *http.Client
}

// NewSlackTrigger returns a new Slack trigger context
func NewSlackTrigger(k8sClient kubernetes.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger, httpClient *http.Client) (*SlackTrigger, error) {
	return &SlackTrigger{
		K8sClient:  k8sClient,
		Sensor:     sensor,
		Trigger:    trigger,
		Logger:     logger,
		httpClient: httpClient,
	}, nil
}

func (t *SlackTrigger) FetchResource() (interface{}, error) {
	return t.Trigger.Template.Slack, nil
}

func (t *SlackTrigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the Slack trigger resource")
	}
	parameters := t.Trigger.Template.Slack.Parameters

	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, t.Trigger.Template.Slack.Parameters, triggers.ExtractEvents(sensor, parameters))
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
func (t *SlackTrigger) Execute(resource interface{}) (interface{}, error) {
	t.Logger.Infoln("executing SlackTrigger")
	_, ok := resource.(*v1alpha1.SlackTrigger)
	if !ok {
		return nil, errors.New("failed to marshal the Slack trigger resource")
	}

	slacktrigger := t.Trigger.Template.Slack

	channel := slacktrigger.Channel
	if channel == "" {
		return nil, errors.New("no slack channel provided")
	}

	message := slacktrigger.Message
	if message == "" {
		return nil, errors.New("no slack message to post")
	}

	slackToken := ""
	if slacktrigger.SlackToken != nil {
		secretToken, err := common.GetSecrets(t.K8sClient, slacktrigger.Namespace, slacktrigger.SlackToken)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve the slack token from secret %s and namespace %s", slacktrigger.SlackToken.Name, slacktrigger.Namespace)
		}
		slackToken = secretToken
	}

	api := slack.New(slackToken, slack.OptionDebug(true))
	_, err := api.JoinChannel(channel)
	if err != nil {
		t.Logger.WithField("channel", slacktrigger.Channel).Errorf("unable to join channel...")
		return nil, errors.Wrapf(err, "failed to join channel %s", channel)
	}

	t.Logger.WithField("channel", slacktrigger.Channel).Infoln("posting to channel...")

	channelID, timestamp, err := api.PostMessage(channel, slack.MsgOptionText(message, false))

	t.Logger.WithField("message", slacktrigger.Message).WithField("channelID", channelID).WithField("timestamp", timestamp).Infoln("message successfully sent to channelID with timestamp")
	t.Logger.Infoln("finished executing SlackTrigger")
	return nil, nil
}

func (t *SlackTrigger) ApplyPolicy(resource interface{}) error {
	return nil
}