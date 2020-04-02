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
	"fmt"
	"github.com/argoproj/argo-events/common"
	"k8s.io/client-go/kubernetes"

	//"github.com/argoproj/argo-events/common"
	"net/http"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
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
	fmt.Println("Executing SlackTrigger")
	obj, ok := resource.(*v1alpha1.SlackTrigger)
	if !ok {
		return nil, errors.New("failed to marshal the Slack trigger resource")
	}

	if obj.Payload == nil {
		return nil, errors.New("payload parameters are not specified")
	}

	payload, err := triggers.ConstructPayload(t.Sensor, obj.Payload)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Sensor payload: %v\n", payload)

	slacktrigger := t.Trigger.Template.Slack

	channel := slacktrigger.Channel
	slackMessage := slacktrigger.Message
	slackToken := slacktrigger.SlackToken

	secretToken := ""

	if slacktrigger.SecretToken != nil && slackToken == ""{
		secretToken, err = common.GetSecrets(t.K8sClient, slacktrigger.Namespace, slacktrigger.SecretToken)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve the username from secret %s and namespace %s", slacktrigger.SecretToken.Name, slacktrigger.Namespace)
		}
		slackToken = secretToken
	}


	api := slack.New(slackToken, slack.OptionDebug(true))
	joinedChannel, err := api.JoinChannel(channel)
	fmt.Printf("joinedChannel: %v\n", joinedChannel)

	t.Logger.WithField("channel", slacktrigger.Channel).Infoln("posting to channel...")
	channelID, timestamp, err := api.PostMessage(channel, slack.MsgOptionText(slackMessage, false))

	//attachment := slack.Attachment{
	//	Pretext: "attachment pretext",
	//	Text:    "attachment text",
	//	//Uncomment the following part to send a field too
	//
	//		Fields: []slack.AttachmentField{
	//			slack.AttachmentField{
	//				Title: "field title",
	//				Value: "field value",
	//			},
	//		},
	//
	//		}
	//channelID, timestamp, err := api.PostMessage(channel, slack.MsgOptionAttachments(attachment))


	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
	fmt.Printf("Message successfully sent to channel %s at %s\n", channelID, timestamp)

	fmt.Println("Finished Executing SlackTrigger")

	return nil, nil
}

// ApplyPolicy applies a policy on trigger execution response if any
func (t *SlackTrigger) ApplyPolicy(resource interface{}) error {
	if t.Trigger.Policy == nil || t.Trigger.Policy.Status == nil || t.Trigger.Policy.Status.Allow == nil {
		return nil
	}
	response, ok := resource.(*http.Response)
	if !ok {
		return errors.New("failed to interpret the trigger execution response")
	}

	p := policy.NewStatusPolicy(response.StatusCode, t.Trigger.Policy.Status.Allow)

	return p.ApplyPolicy()
}