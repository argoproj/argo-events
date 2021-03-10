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
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"go.uber.org/zap"
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
func (router *Router) getCredentials(keySelector *corev1.SecretKeySelector) (*cred, error) {
	token, err := common.GetSecretFromVolume(keySelector)
	if err != nil {
		return nil, errors.Wrap(err, "token not founnd")
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

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
	)

	logger.Info("received a request, processing it...")

	if !route.Active {
		logger.Info("endpoint is not active, won't process the request")
		common.SendErrorResponse(writer, "inactive endpoint")
		return
	}

	startTime := time.Now()
	defer func(start time.Time) {
		elapsed := time.Since(start)
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(elapsed/time.Millisecond))
	}(startTime)

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Errorw("failed to parse request body", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	event := &events.GitLabEventData{
		Headers:  request.Header,
		Body:     (*json.RawMessage)(&body),
		Metadata: router.gitlabEventSource.Metadata,
	}

	eventBody, err := json.Marshal(event)
	if err != nil {
		logger.Info("failed to marshal event")
		common.SendErrorResponse(writer, "invalid event")
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	logger.Info("dispatching event on route's data channel")
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

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
		"project-id", gitlabEventSource.ProjectID,
	)

	logger.Info("retrieving the access token credentials...")
	c, err := router.getCredentials(gitlabEventSource.AccessToken)
	if err != nil {
		return errors.Errorf("failed to get gitlab credentials. err: %+v", err)
	}

	logger.Info("setting up the client to connect to GitLab...")
	router.gitlabClient, err = gitlab.NewClient(c.token, gitlab.WithBaseURL(gitlabEventSource.GitlabBaseURL))
	if err != nil {
		return errors.Wrapf(err, "failed to initialize client")
	}

	formattedURL := common.FormattedURL(gitlabEventSource.Webhook.URL, gitlabEventSource.Webhook.Endpoint)

	hooks, _, err := router.gitlabClient.Projects.ListProjectHooks(gitlabEventSource.ProjectID, &gitlab.ListProjectHooksOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to list existing hooks to check for duplicates for project id %s", router.gitlabEventSource.ProjectID)
	}

	var existingHook *gitlab.ProjectHook
	isAlreadyExists := false

	for _, hook := range hooks {
		if hook.URL == formattedURL {
			existingHook = hook
			isAlreadyExists = true
		}
	}

	defaultEventValue := false

	editOpt := &gitlab.EditProjectHookOptions{
		URL:                      &formattedURL,
		ConfidentialNoteEvents:   &defaultEventValue,
		PushEvents:               &defaultEventValue,
		IssuesEvents:             &defaultEventValue,
		ConfidentialIssuesEvents: &defaultEventValue,
		MergeRequestsEvents:      &defaultEventValue,
		TagPushEvents:            &defaultEventValue,
		NoteEvents:               &defaultEventValue,
		JobEvents:                &defaultEventValue,
		PipelineEvents:           &defaultEventValue,
		WikiPageEvents:           &defaultEventValue,
		EnableSSLVerification:    &router.gitlabEventSource.EnableSSLVerification,
		Token:                    &c.token,
	}

	addOpt := &gitlab.AddProjectHookOptions{
		URL:                      &formattedURL,
		Token:                    &c.token,
		EnableSSLVerification:    &router.gitlabEventSource.EnableSSLVerification,
		ConfidentialNoteEvents:   &defaultEventValue,
		PushEvents:               &defaultEventValue,
		IssuesEvents:             &defaultEventValue,
		ConfidentialIssuesEvents: &defaultEventValue,
		MergeRequestsEvents:      &defaultEventValue,
		TagPushEvents:            &defaultEventValue,
		NoteEvents:               &defaultEventValue,
		JobEvents:                &defaultEventValue,
		PipelineEvents:           &defaultEventValue,
		WikiPageEvents:           &defaultEventValue,
	}

	var opt interface{}

	opt = addOpt
	if isAlreadyExists {
		opt = editOpt
	}

	logger.Info("configuring the GitLab events for the hook...")

	for _, event := range gitlabEventSource.Events {
		elem := reflect.ValueOf(opt).Elem().FieldByName(event)
		if ok := elem.IsValid(); !ok {
			return errors.Errorf("unknown event %s", event)
		}

		iev := reflect.New(elem.Type().Elem())
		reflect.Indirect(iev).SetBool(true)
		elem.Set(iev)
	}

	var newHook *gitlab.ProjectHook

	if !isAlreadyExists {
		logger.Info("creating project hook...")
		newHook, _, err = router.gitlabClient.Projects.AddProjectHook(router.gitlabEventSource.ProjectID, opt.(*gitlab.AddProjectHookOptions))
		if err != nil {
			return errors.Errorf("failed to add project hook. err: %+v", err)
		}
	} else {
		logger.Info("project hook already exists, updating it...")
		if existingHook == nil {
			return errors.Errorf("existing hook contents are empty, unable to edit existing webhook")
		}
		newHook, _, err = router.gitlabClient.Projects.EditProjectHook(router.gitlabEventSource.ProjectID, existingHook.ID, opt.(*gitlab.EditProjectHookOptions))
		if err != nil {
			return errors.Errorf("failed to add project hook. err: %+v", err)
		}
	}

	router.hook = newHook
	logger.With("hook-id", newHook.ID).Info("hook registered for the project")
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (router *Router) PostInactivate() error {
	gitlabEventSource := router.gitlabEventSource
	route := router.route

	if gitlabEventSource.DeleteHookOnFinish {
		logger := route.Logger.With(
			"project-id", gitlabEventSource.ProjectID,
			"hook-id", router.hook.ID,
		)

		logger.Info("deleting project hook...")
		if _, err := router.gitlabClient.Projects.DeleteProjectHook(router.gitlabEventSource.ProjectID, router.hook.ID); err != nil {
			return errors.Errorf("failed to delete hook. err: %+v", err)
		}

		logger.Info("gitlab hook deleted")
	}
	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	logger := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	logger.Info("started processing the Gitlab event source...")

	defer sources.Recover(el.GetEventName())

	gitlabEventSource := &el.GitlabEventSource

	route := webhook.NewRoute(gitlabEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)

	return webhook.ManageRoute(ctx, &Router{
		route:             route,
		gitlabEventSource: gitlabEventSource,
	}, controller, dispatch)
}
