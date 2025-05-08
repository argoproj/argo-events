/*
Copyright 2018 The Argoproj Authors.

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

package gerrit

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	gerrit "github.com/andygrunwald/go-gerrit"
	"go.uber.org/zap"

	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
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
		sharedutil.SendErrorResponse(writer, "inactive endpoint")
		return
	}

	defer func(start time.Time) {
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	request.Body = http.MaxBytesReader(writer, request.Body, route.Context.GetMaxPayloadSize())
	body, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Errorw("failed to parse request body", zap.Error(err))
		sharedutil.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	event := &events.GerritEventData{
		Headers:  request.Header,
		Body:     (*json.RawMessage)(&body),
		Metadata: router.gerritEventSource.Metadata,
	}

	eventBody, err := json.Marshal(event)
	if err != nil {
		logger.Info("failed to marshal event")
		sharedutil.SendErrorResponse(writer, "invalid event")
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	webhook.DispatchEvent(route, eventBody, logger, writer)
}

// PostActivate performs operations once the route is activated and ready to consume requests
func (router *Router) PostActivate() error {
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (router *Router) PostInactivate() error {
	gerritEventSource := router.gerritEventSource
	if !gerritEventSource.NeedToCreateHooks() || !gerritEventSource.DeleteHookOnFinish {
		return nil
	}

	logger := router.route.Logger
	logger.Info("deleting Gerrit hooks...")

	for _, p := range gerritEventSource.Projects {
		_, ok := router.projectHooks[p]
		if !ok {
			return fmt.Errorf("can not find hook ID for project %s", p)
		}
		if err := router.gerritHookService.Delete(p, gerritEventSource.HookName); err != nil {
			return fmt.Errorf("failed to delete hook for project %s. err: %w", p, err)
		}
		logger.Infof("Gerrit hook deleted for project %s", p)
	}
	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	logger := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	logger.Info("started processing the Gerrit event source...")

	defer sources.Recover(el.GetEventName())

	gerritEventSource := &el.GerritEventSource

	route := webhook.NewRoute(gerritEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	router := &Router{
		route:             route,
		gerritEventSource: gerritEventSource,
		projectHooks:      make(map[string]string),
	}

	if gerritEventSource.NeedToCreateHooks() {
		// In order to set up a hook for the Gerrit project,
		// 1. Set up Gerrit client with basic auth
		// 2. Configure Hook with given event type
		// 3. Create project hook

		logger.Info("retrieving the access token credentials...")

		formattedURL := sharedutil.FormattedURL(gerritEventSource.Webhook.URL, gerritEventSource.Webhook.Endpoint)
		opt := &ProjectHookConfigs{
			URL:       formattedURL,
			Events:    router.gerritEventSource.Events,
			SslVerify: router.gerritEventSource.SslVerify,
		}

		logger.Info("setting up the client to connect to Gerrit...")
		var err error
		router.gerritClient, err = gerrit.NewClient(router.gerritEventSource.GerritBaseURL, nil)
		if err != nil {
			return fmt.Errorf("failed to initialize client, %w", err)
		}
		if gerritEventSource.Auth != nil {
			username, err := sharedutil.GetSecretFromVolume(gerritEventSource.Auth.Username)
			if err != nil {
				return fmt.Errorf("username not found, %w", err)
			}
			password, err := sharedutil.GetSecretFromVolume(gerritEventSource.Auth.Password)
			if err != nil {
				return fmt.Errorf("password not found, %w", err)
			}
			router.gerritClient.Authentication.SetBasicAuth(username, password)
		}
		router.gerritHookService = newGerritWebhookService(router.gerritClient)

		f := func() {
			for _, p := range gerritEventSource.Projects {
				hooks, err := router.gerritHookService.List(p)
				if err != nil {
					logger.Errorf("failed to list existing webhooks of project %s. err: %+v", p, err)
					continue
				}
				// hook already exist
				if h, ok := hooks[gerritEventSource.HookName]; ok {
					if h.URL == formattedURL {
						router.projectHooks[p] = gerritEventSource.HookName
						continue
					}
				}
				logger.Infof("hook not found for project %s, creating ...", p)
				if _, err := router.gerritHookService.Create(p, gerritEventSource.HookName, opt); err != nil {
					logger.Errorf("failed to create gerrit webhook for project %s. err: %+v", p, err)
					continue
				}
				router.projectHooks[p] = gerritEventSource.HookName
				time.Sleep(500 * time.Millisecond)
			}
		}

		// Mitigate race conditions - it might create multiple hooks with same config when replicas > 1
		randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(2000)))
		time.Sleep(time.Duration(randomNum.Int64()) * time.Millisecond)
		f()

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			// Another kind of race conditions might happen when pods do rolling upgrade - new pod starts
			// and old pod terminates, if DeleteHookOnFinish is true, the hook will be deleted from gerrit.
			// This is a workaround to mitigate the race conditions.
			logger.Info("starting gerrit hooks manager daemon")
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					logger.Info("exiting gerrit hooks manager daemon")
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
