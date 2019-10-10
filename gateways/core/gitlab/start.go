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

var (
	helper = gwcommon.NewWebhookHelper()
)

func init() {
	go gwcommon.InitRouteChannels(helper)
}

// getCredentials for gitlab
func (rc *RouteConfig) getCredentials(gs *GitlabSecret) (*cred, error) {
	token, err := store.GetSecrets(rc.clientset, rc.namespace, gs.Name, gs.Key)
	if err != nil {
		return nil, err
	}
	return &cred{
		token: token,
	}, nil
}

func (rc *RouteConfig) GetRoute() *gwcommon.Route {
	return rc.route
}

func (rc *RouteConfig) PostStart() error {
	c, err := rc.getCredentials(rc.ges.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get gitlab credentials. err: %+v", err)
	}

	rc.client = gitlab.NewClient(nil, c.token)
	if err = rc.client.SetBaseURL(rc.ges.GitlabBaseURL); err != nil {
		return fmt.Errorf("failed to set gitlab base url, err: %+v", err)
	}

	formattedUrl := gwcommon.GenerateFormattedURL(rc.ges.Hook)

	opt := &gitlab.AddProjectHookOptions{
		URL:                   &formattedUrl,
		Token:                 &c.token,
		EnableSSLVerification: &rc.ges.EnableSSLVerification,
	}

	elem := reflect.ValueOf(opt).Elem().FieldByName(string(rc.ges.Event))
	if ok := elem.IsValid(); !ok {
		return fmt.Errorf("unknown event %s", rc.ges.Event)
	}

	iev := reflect.New(elem.Type().Elem())
	reflect.Indirect(iev).SetBool(true)
	elem.Set(iev)

	hook, _, err := rc.client.Projects.AddProjectHook(rc.ges.ProjectId, opt)
	if err != nil {
		return fmt.Errorf("failed to add project hook. err: %+v", err)
	}

	rc.hook = hook
	rc.route.Logger.WithField(common.LabelEventSource, rc.route.EventSource.Name).Info("gitlab hook created")
	return nil
}

func (rc *RouteConfig) PostStop() error {
	if _, err := rc.client.Projects.DeleteProjectHook(rc.ges.ProjectId, rc.hook.ID); err != nil {
		return fmt.Errorf("failed to delete hook. err: %+v", err)
	}
	rc.route.Logger.WithField(common.LabelEventSource, rc.route.EventSource.Name).Info("gitlab hook deleted")
	return nil
}

// routeActiveHandler handles new route
func (rc *RouteConfig) RouteHandler(writer http.ResponseWriter, request *http.Request) {
	r := rc.route

	log := r.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: r.EventSource.Name,
			common.LabelEndpoint:    r.Webhook.Endpoint,
			common.LabelPort:        r.Webhook.Port,
		})

	log.Info("request received")

	if !helper.ActiveEndpoints[r.Webhook.Endpoint].Active {
		log.Info("endpoint is not active")
		common.SendErrorResponse(writer, "")
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.WithError(err).Error("failed to parse request body")
		common.SendErrorResponse(writer, "")
		return
	}

	helper.ActiveEndpoints[r.Webhook.Endpoint].DataCh <- body
	log.Info("request successfully processed")
	common.SendSuccessResponse(writer, "")
}

// StartEventSource starts an event source
func (ese *GitlabEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	log := ese.Log.WithField(common.LabelEventSource, eventSource.Name)

	log.Info("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}
	gl := config.(*gitlabEventSource)

	return gwcommon.ProcessRoute(&RouteConfig{
		route: &gwcommon.Route{
			EventSource: eventSource,
			Logger:      ese.Log,
			Webhook:     gl.Hook,
			StartCh:     make(chan struct{}),
		},
		namespace: ese.Namespace,
		clientset: ese.Clientset,
		ges:       gl,
	}, helper, eventStream)
}
