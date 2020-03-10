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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/argoproj/argo-events/store"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	corev1 "k8s.io/api/core/v1"
)

// controller controls the webhook operations
var (
	controller = webhook.NewController()
)

// set up the activation and inactivation channels to control the state of routes.
func init() {
	go webhook.ProcessRouteStatus(controller)
}

// getCredentials retrieves credentials to connect to GitLab
func (router *Router) getCredentials(keySelector *corev1.SecretKeySelector, namespace string) (*cred, error) {
	token, err := store.GetSecrets(router.k8sClient, namespace, keySelector.Name, keySelector.Key)
	if err != nil {
		return nil, err
	}
	return &cred{
		token: token,
	}, nil
}

// Implement Router
// 1. GetRoute
// 2. HandleRoute
// 3. PostActivate
// 4. PostDeactivate

// GetRoute returns the route
func (router *Router) GetRoute() *webhook.Route {
	return router.route
}

// HandleRoute handles incoming requests on the route
func (router *Router) HandleRoute(writer http.ResponseWriter, request *http.Request) {
	route := router.GetRoute()

	logger := route.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: route.EventSource.Name,
			common.LabelEndpoint:    route.Context.Endpoint,
			common.LabelPort:        route.Context.Port,
		})

	logger.Info("received a request, processing it...")

	if !route.Active {
		logger.Info("endpoint is not active, won't process the request")
		common.SendErrorResponse(writer, "inactive endpoint")
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.WithError(err).Error("failed to parse request body")
		common.SendErrorResponse(writer, err.Error())
		return
	}

	event := &events.GitLabEventData{
		Headers: request.Header,
		Body:    (*json.RawMessage)(&body),
	}

	eventBody, err := json.Marshal(event)
	if err != nil {
		logger.Info("failed to marshal event")
		common.SendErrorResponse(writer, "invalid event")
		return
	}

	logger.Infoln("dispatching event on route's data channel")
	route.DataCh <- eventBody

	logger.Info("request successfully processed")
	common.SendSuccessResponse(writer, "success")
}

// PostActivate performs operations once the route is activated and ready to consume requests
func (router *Router) PostActivate() error {
	route := router.GetRoute()
	gitlabEventSource := router.gitlabEventSource

	// In order to set up a hook for the GitLab project,
	// 1. Get the API access token for client
	// 2. Set up GitLab client
	// 3. Configure Hook with given event type
	// 4. Create project hook

	logger := route.Logger.WithFields(map[string]interface{}{
		common.LabelEventSource: route.EventSource.Name,
		"event-type":            gitlabEventSource.Event,
		"project-id":            gitlabEventSource.ProjectId,
	})

	logger.Infoln("retrieving the access token credentials...")
	c, err := router.getCredentials(gitlabEventSource.AccessToken, gitlabEventSource.Namespace)
	if err != nil {
		return errors.Errorf("failed to get gitlab credentials. err: %+v", err)
	}

	logger.Infoln("setting up the client to connect to GitLab...")
	router.gitlabClient = gitlab.NewClient(nil, c.token)
	if err = router.gitlabClient.SetBaseURL(gitlabEventSource.GitlabBaseURL); err != nil {
		return errors.Errorf("failed to set gitlab base url, err: %+v", err)
	}

	formattedUrl := common.FormattedURL(gitlabEventSource.Webhook.URL, gitlabEventSource.Webhook.Endpoint)

	// Get existing webhooks and check if the integration for same url and event type is already available
	if !gitlabEventSource.AllowDuplicate {
		hooks, _, err := router.gitlabClient.Projects.ListProjectHooks(router.gitlabEventSource.ProjectId, &gitlab.ListProjectHooksOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to list existing hooks to check for duplicates for project id %s", router.gitlabEventSource.ProjectId)
		}

		for _, hook := range hooks {
			elem := reflect.ValueOf(hook).Elem().FieldByName(router.gitlabEventSource.Event)
			if ok := elem.IsValid(); !ok {
				return errors.Errorf("unknown event %s", router.gitlabEventSource.Event)
			}
			value := elem.Bool()

			if value && hook.URL == formattedUrl {
				logger.Infoln("webhook already exists, won't register it...")
				return nil
			}
		}
	}

	opt := &gitlab.AddProjectHookOptions{
		URL:                   &formattedUrl,
		Token:                 &c.token,
		EnableSSLVerification: &router.gitlabEventSource.EnableSSLVerification,
	}

	logger.Infoln("configuring the type of the GitLab event the hook must register against...")
	elem := reflect.ValueOf(opt).Elem().FieldByName(router.gitlabEventSource.Event)
	if ok := elem.IsValid(); !ok {
		return errors.Errorf("unknown event %s", router.gitlabEventSource.Event)
	}

	iev := reflect.New(elem.Type().Elem())
	reflect.Indirect(iev).SetBool(true)
	elem.Set(iev)

	logger.Infoln("creating project hook...")
	hook, _, err := router.gitlabClient.Projects.AddProjectHook(router.gitlabEventSource.ProjectId, opt)
	if err != nil {
		return errors.Errorf("failed to add project hook. err: %+v", err)
	}

	router.hook = hook
	logger.WithField("hook-id", hook.ID).Info("hook created for the project")
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (router *Router) PostInactivate() error {
	gitlabEventSource := router.gitlabEventSource
	route := router.route

	if gitlabEventSource.DeleteHookOnFinish {
		logger := route.Logger.WithFields(map[string]interface{}{
			common.LabelEventSource: route.EventSource.Name,
			"project-id":            gitlabEventSource.ProjectId,
			"hook-id":               router.hook.ID,
		})

		logger.Infoln("deleting project hook...")
		if _, err := router.gitlabClient.Projects.DeleteProjectHook(router.gitlabEventSource.ProjectId, router.hook.ID); err != nil {
			return errors.Errorf("failed to delete hook. err: %+v", err)
		}

		logger.Infoln("gitlab hook deleted")
	}
	return nil
}

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer server.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Info("started processing the event source...")

	var gitlabEventSource *v1alpha1.GitlabEventSource
	if err := yaml.Unmarshal(eventSource.Value, &gitlabEventSource); err != nil {
		logger.WithError(err).Error("failed to parse the event source")
		return err
	}

	route := webhook.NewRoute(gitlabEventSource.Webhook, listener.Logger, eventSource)

	return webhook.ManageRoute(&Router{
		route:             route,
		k8sClient:         listener.K8sClient,
		gitlabEventSource: gitlabEventSource,
	}, controller, eventStream)
}
