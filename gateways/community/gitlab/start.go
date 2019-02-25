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

package gitlab

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/store"
	"github.com/xanzy/go-gitlab"
)

const (
	LabelGitlabConfig = "config"
	LabelGitlabClient = "client"
	LabelWebhook      = "hook"
)

var (
	helper = gwcommon.NewWebhookHelper()
)

func init() {
	go gwcommon.InitRouteChannels(helper)
}

// getCredentials for gitlab
func (ese *GitlabEventSourceExecutor) getCredentials(gs *GitlabSecret) (*cred, error) {
	token, err := store.GetSecrets(ese.Clientset, ese.Namespace, gs.Name, gs.Key)
	if err != nil {
		return nil, err
	}
	return &cred{
		token: token,
	}, nil
}

func (ese *GitlabEventSourceExecutor) PostActivate(rc *gwcommon.RouteConfig) error {
	gl := rc.Configs[LabelGitlabConfig].(*glab)

	c, err := ese.getCredentials(gl.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get gitlab credentials. err: %+v", err)
	}

	client := gitlab.NewClient(nil, c.token)
	if err = client.SetBaseURL(gl.GitlabBaseURL); err != nil {
		return fmt.Errorf("failed to set gitlab base url, err: %+v", err)
	}

	rc.Configs[LabelGitlabClient] = client

	opt := &gitlab.AddProjectHookOptions{
		URL:   &gl.URL,
		Token: &c.token,
		EnableSSLVerification: &gl.EnableSSLVerification,
	}

	elem := reflect.ValueOf(opt).Elem().FieldByName(string(gl.Event))
	if ok := elem.IsValid(); !ok {
		return fmt.Errorf("unknown event %s", gl.Event)
	}

	iev := reflect.New(elem.Type().Elem())
	reflect.Indirect(iev).SetBool(true)
	elem.Set(iev)

	hook, _, err := client.Projects.AddProjectHook(gl.ProjectId, opt)
	if err != nil {
		return fmt.Errorf("failed to add project hook. err: %+v", err)
	}
	rc.Configs[LabelWebhook] = hook

	rc.Log.Info().Str("event-source-name", rc.EventSource.Name).Interface("hook-id", hook.ID).Msg("gitlab hook created")
	return nil
}

func PostStop(rc *gwcommon.RouteConfig) error {
	gl := rc.Configs[LabelGitlabConfig].(*glab)
	client := rc.Configs[LabelGitlabClient].(*gitlab.Client)
	hook := rc.Configs[LabelWebhook].(*gitlab.ProjectHook)

	if _, err := client.Projects.DeleteProjectHook(gl.ProjectId, hook.ID); err != nil {
		rc.Log.Error().Err(err).Str("event-source-name", rc.EventSource.Name).Interface("hook-id", hook.ID).Msg("failed to delete gitlab hook")
		return err
	}
	rc.Log.Info().Str("event-source-name", rc.EventSource.Name).Interface("hook-id", hook.ID).Msg("gitlab hook deleted")
	return nil
}

// routeActiveHandler handles new route
func RouteActiveHandler(writer http.ResponseWriter, request *http.Request, rc *gwcommon.RouteConfig) {
	var response string

	logger := rc.Log.With().Str("event-source", rc.EventSource.Name).Str("endpoint", rc.Webhook.Endpoint).
		Str("port", rc.Webhook.Port).
		Str("http-method", request.Method).Logger()
	logger.Info().Msg("request received")

	if !helper.ActiveEndpoints[rc.Webhook.Endpoint].Active {
		response = fmt.Sprintf("the route: endpoint %s and method %s is deactived", rc.Webhook.Endpoint, rc.Webhook.Method)
		logger.Info().Msg("endpoint is not active")
		common.SendErrorResponse(writer, response)
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse request body")
		common.SendErrorResponse(writer, fmt.Sprintf("failed to parse request. err: %+v", err))
		return
	}

	helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh <- body
	response = "request successfully processed"
	logger.Info().Msg(response)
	common.SendSuccessResponse(writer, response)
}

// StartEventSource starts an event source
func (ese *GitlabEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		ese.Log.Error().Err(err).Str("event-source-name", eventSource.Name).Msg("failed to parse event source")
		return err
	}
	gl := config.(*glab)

	return gwcommon.ProcessRoute(&gwcommon.RouteConfig{
		Webhook: &gwcommon.Webhook{
			Endpoint: gwcommon.FormatWebhookEndpoint(gl.Endpoint),
			Port:     gl.Port,
		},
		Configs: map[string]interface{}{
			LabelGitlabConfig: gl,
		},
		Log:                ese.Log,
		EventSource:        eventSource,
		PostActivate:       ese.PostActivate,
		PostStop:           PostStop,
		RouteActiveHandler: RouteActiveHandler,
		StartCh:            make(chan struct{}),
	}, helper, eventStream)
}
