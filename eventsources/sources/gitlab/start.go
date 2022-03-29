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
	"math/rand"
	"net/http"
	"reflect"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"go.uber.org/zap"
)

// controller controls the webhook operations
var (
	controller = webhook.NewController()
)

// set up the activation and inactivation channels to control the state of routes.
func init() {
	go webhook.ProcessRouteStatus(controller)
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

	defer func(start time.Time) {
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	if router.secretToken != "" {
		if t := request.Header.Get("X-Gitlab-Token"); t != router.secretToken {
			common.SendErrorResponse(writer, "token mismatch")
			return
		}
	}

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
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (router *Router) PostInactivate() error {
	gitlabEventSource := router.gitlabEventSource
	if !gitlabEventSource.NeedToCreateHooks() || !gitlabEventSource.DeleteHookOnFinish {
		return nil
	}

	logger := router.route.Logger
	logger.Info("deleting Gitlab hooks...")

	for _, p := range gitlabEventSource.GetProjects() {
		id, ok := router.hookIDs[p]
		if !ok {
			return errors.Errorf("can not find hook ID for project %s", p)
		}
		if _, err := router.gitlabClient.Projects.DeleteProjectHook(p, id); err != nil {
			return errors.Errorf("failed to delete hook for project %s. err: %+v", p, err)
		}
		logger.Infof("Gitlab hook deleted for project %s", p)
	}
	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Options) error) error {
	logger := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	logger.Info("started processing the Gitlab event source...")

	defer sources.Recover(el.GetEventName())

	gitlabEventSource := &el.GitlabEventSource

	route := webhook.NewRoute(gitlabEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	router := &Router{
		route:             route,
		gitlabEventSource: gitlabEventSource,
	}

	if gitlabEventSource.NeedToCreateHooks() {
		// In order to set up a hook for the GitLab project,
		// 1. Get the API access token for client
		// 2. Set up GitLab client
		// 3. Configure Hook with given event type
		// 4. Create project hook

		logger.Info("retrieving the access token credentials...")

		defaultEventValue := false
		formattedURL := common.FormattedURL(gitlabEventSource.Webhook.URL, gitlabEventSource.Webhook.Endpoint)
		opt := &gitlab.AddProjectHookOptions{
			URL:                      &formattedURL,
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

		for _, event := range gitlabEventSource.Events {
			elem := reflect.ValueOf(opt).Elem().FieldByName(event)
			if ok := elem.IsValid(); !ok {
				return errors.Errorf("unknown event %s", event)
			}
			iev := reflect.New(elem.Type().Elem())
			reflect.Indirect(iev).SetBool(true)
			elem.Set(iev)
		}

		if gitlabEventSource.SecretToken != nil {
			token, err := common.GetSecretFromVolume(gitlabEventSource.SecretToken)
			if err != nil {
				return errors.Errorf("failed to retrieve secret token. err: %+v", err)
			}
			opt.Token = &token
			router.secretToken = token
		}

		accessToken, err := common.GetSecretFromVolume(gitlabEventSource.AccessToken)
		if err != nil {
			return errors.Errorf("failed to get gitlab credentials. err: %+v", err)
		}

		logger.Info("setting up the client to connect to GitLab...")
		router.gitlabClient, err = gitlab.NewClient(accessToken, gitlab.WithBaseURL(gitlabEventSource.GitlabBaseURL))
		if err != nil {
			return errors.Wrapf(err, "failed to initialize client")
		}

		getHook := func(hooks []*gitlab.ProjectHook, url string, events []string) *gitlab.ProjectHook {
			for _, h := range hooks {
				if h.URL != url {
					continue
				}
				return h
			}
			return nil
		}

		router.hookIDs = make(map[string]int)

		f := func() {
			for _, p := range gitlabEventSource.GetProjects() {
				hooks, _, err := router.gitlabClient.Projects.ListProjectHooks(p, &gitlab.ListProjectHooksOptions{})
				if err != nil {
					logger.Errorf("failed to list existing webhooks of project %s. err: %+v", p, err)
					continue
				}
				hook := getHook(hooks, formattedURL, gitlabEventSource.Events)
				if hook != nil {
					router.hookIDs[p] = hook.ID
					continue
				}
				logger.Infof("hook not found for project %s, creating ...", p)
				hook, _, err = router.gitlabClient.Projects.AddProjectHook(p, opt)
				if err != nil {
					logger.Errorf("failed to create gitlab webhook for project %s. err: %+v", p, err)
					continue
				}
				router.hookIDs[p] = hook.ID
				time.Sleep(500 * time.Millisecond)
			}
		}

		// Mitigate race condtions - it might create multiple hooks with same config when replicas > 1
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		time.Sleep(time.Duration(r1.Intn(2000)) * time.Millisecond)
		f()

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			// Another kind of race conditions might happen when pods do rolling upgrade - new pod starts
			// and old pod terminates, if DeleteHookOnFinish is true, the hook will be deleted from gitlab.
			// This is a workround to mitigate the race conditions.
			logger.Info("starting gitlab hooks manager daemon")
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					logger.Info("exiting gitlab hooks manager daemon")
					return
				case <-ticker.C:
					f()
				}
			}
		}()
	} else {
		logger.Info("no need to create webhooks")
	}

	return webhook.ManageRoute(ctx, router, controller, dispatch)
}
