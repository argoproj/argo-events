/*

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

package bitbucketserver

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/mitchellh/mapstructure"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/pkg/errors"
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

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Errorw("failed to parse request body", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	event := &events.BitbucketServerEventData{
		Headers:  request.Header,
		Body:     (*json.RawMessage)(&body),
		Metadata: router.bitbucketserverEventSource.Metadata,
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
	bitbucketserverEventSource := router.bitbucketserverEventSource
	route := router.route
	logger := route.Logger

	if bitbucketserverEventSource.DeleteHookOnFinish && len(router.hookIDs) > 0 {
		logger.Info("deleting webhooks from bitbucket")

		bitbucketToken, err := common.GetSecretFromVolume(bitbucketserverEventSource.AccessToken)
		if err != nil {
			return errors.Errorf("failed to get bitbucketserver token. err: %+v", err)
		}

		bitbucketConfig := bitbucketv1.NewConfiguration(bitbucketserverEventSource.BitbucketServerBaseURL)
		bitbucketConfig.AddDefaultHeader("x-atlassian-token", "no-check")
		bitbucketConfig.AddDefaultHeader("x-requested-with", "XMLHttpRequest")

		for _, repo := range bitbucketserverEventSource.GetBitbucketServerRepositories() {
			id, ok := router.hookIDs[repo.ProjectKey+","+repo.RepositorySlug]
			if !ok {
				return errors.Errorf("can not find hook ID for project-key: %s, repository-slug: %s", repo.ProjectKey, repo.RepositorySlug)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			ctx = context.WithValue(ctx, bitbucketv1.ContextAccessToken, bitbucketToken)
			bitbucketClient := bitbucketv1.NewAPIClient(ctx, bitbucketConfig)

			_, err = bitbucketClient.DefaultApi.DeleteWebhook(repo.ProjectKey, repo.RepositorySlug, int32(id))
			if err != nil {
				return errors.Errorf("failed to delete bitbucketserver webhook. err: %+v", err)
			}

			logger.Infow("bitbucket server webhook deleted",
				zap.String("project-key", repo.ProjectKey), zap.String("repository-slug", repo.RepositorySlug))
		}
	} else {
		logger.Info("no need to delete webhooks, skipping.")
	}

	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Options) error) error {
	defer sources.Recover(el.GetEventName())

	bitbucketserverEventSource := &el.BitbucketServerEventSource

	logger := logging.FromContext(ctx).With(
		logging.LabelEventSourceType, el.GetEventSourceType(),
		logging.LabelEventName, el.GetEventName(),
		"base-url", bitbucketserverEventSource.BitbucketServerBaseURL,
	)

	logger.Info("started processing the Bitbucket Server event source...")

	route := webhook.NewRoute(bitbucketserverEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	router := &Router{
		route:                      route,
		bitbucketserverEventSource: bitbucketserverEventSource,
		hookIDs:                    make(map[string]int),
	}

	logger.Info("retrieving the access token credentials...")
	bitbucketToken, err := common.GetSecretFromVolume(bitbucketserverEventSource.AccessToken)
	if err != nil {
		return errors.Errorf("failed to get bitbucketserver token. err: %+v", err)
	}

	logger.Info("setting up the client to connect to Bitbucket Server...")
	bitbucketConfig := bitbucketv1.NewConfiguration(bitbucketserverEventSource.BitbucketServerBaseURL)
	bitbucketConfig.AddDefaultHeader("x-atlassian-token", "no-check")
	bitbucketConfig.AddDefaultHeader("x-requested-with", "XMLHttpRequest")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = context.WithValue(ctx, bitbucketv1.ContextAccessToken, bitbucketToken)

	createWebhooks := func() {
		for _, repo := range bitbucketserverEventSource.GetBitbucketServerRepositories() {
			if err = router.CreateBitbucketWebhook(ctx, bitbucketConfig, repo); err != nil {
				logger.Errorw("failed to create/update Bitbucket webhook",
					zap.String("project-key", repo.ProjectKey), zap.String("repository-slug", repo.RepositorySlug), zap.Error(err))
				continue
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	// When running multiple replicas of the eventsource, they will all try to create the webhook.
	// Randomly sleep some time to mitigate the issue.
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	time.Sleep(time.Duration(r1.Intn(2000)) * time.Millisecond)
	createWebhooks()

	go func() {
		// Another kind of race conditions might happen when pods do rolling upgrade - new pod starts
		// and old pod terminates, if DeleteHookOnFinish is true, the hook will be deleted from Bitbucket.
		// This is a workaround to mitigate the race conditions.
		logger.Info("starting bitbucket hooks manager daemon")
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("exiting bitbucket hooks manager daemon")
				return
			case <-ticker.C:
				createWebhooks()
			}
		}
	}()

	return webhook.ManageRoute(ctx, router, controller, dispatch)
}

func (router *Router) CreateBitbucketWebhook(ctx context.Context, bitbucketConfig *bitbucketv1.Configuration, repo v1alpha1.BitbucketServerRepository) error {
	bitbucketserverEventSource := router.bitbucketserverEventSource
	route := router.route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
		"project-key", repo.ProjectKey,
		"repository-slug", repo.RepositorySlug,
		"base-url", bitbucketserverEventSource.BitbucketServerBaseURL,
	)

	bitbucketClient := bitbucketv1.NewAPIClient(ctx, bitbucketConfig)

	formattedURL := common.FormattedURL(bitbucketserverEventSource.Webhook.URL, bitbucketserverEventSource.Webhook.Endpoint)

	apiResponse, err := bitbucketClient.DefaultApi.FindWebhooks(repo.ProjectKey, repo.RepositorySlug, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to list existing hooks to check for duplicates for repository %s/%s", repo.ProjectKey, repo.RepositorySlug)
	}

	hooks, err := bitbucketv1.GetWebhooksResponse(apiResponse)
	if err != nil {
		return errors.Wrapf(err, "failed to convert the list of webhooks for repository %s/%s", repo.ProjectKey, repo.RepositorySlug)
	}

	var existingHook bitbucketv1.Webhook
	isAlreadyExists := false

	for _, hook := range hooks {
		if hook.Url == formattedURL {
			isAlreadyExists = true
			existingHook = hook
			router.hookIDs[repo.ProjectKey+","+repo.RepositorySlug] = hook.ID
			break
		}
	}

	logger.Info("retrieving the webhook secret...")
	webhookSecret, err := common.GetSecretFromVolume(bitbucketserverEventSource.WebhookSecret)
	if err != nil {
		return errors.Errorf("failed to get bitbucketserver webhook secret. err: %+v", err)
	}

	newHook := bitbucketv1.Webhook{
		Name:          "Argo Events",
		Url:           formattedURL,
		Active:        true,
		Events:        bitbucketserverEventSource.Events,
		Configuration: bitbucketv1.WebhookConfiguration{Secret: webhookSecret},
	}

	localVarPostBody, err := json.Marshal(newHook)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal new webhook to JSON")
	}

	// Create the webhook when it doesn't exist yet
	if !isAlreadyExists {
		apiResponse, err = bitbucketClient.DefaultApi.CreateWebhook(repo.ProjectKey, repo.RepositorySlug, localVarPostBody, []string{"application/json"})
		if err != nil {
			return errors.Errorf("failed to add webhook. err: %+v", err)
		}

		var createdHook *bitbucketv1.Webhook
		err = mapstructure.Decode(apiResponse.Values, &createdHook)
		if err != nil {
			return errors.Errorf("failed to convert API response to Webhook struct. err: %+v", err)
		}

		router.hookIDs[repo.ProjectKey+","+repo.RepositorySlug] = createdHook.ID

		logger.With("hook-id", createdHook.ID).Info("hook successfully registered")

		return nil
	}

	// Update the webhook when it does exist and the configuration has changed
	if isAlreadyExists && (!reflect.DeepEqual(existingHook.Events, newHook.Events) || !reflect.DeepEqual(existingHook.Configuration, newHook.Configuration)) {
		logger.Info("webhook already exists and configuration has changed. Updating webhook.")

		_, err = bitbucketClient.DefaultApi.UpdateWebhook(repo.ProjectKey, repo.RepositorySlug, int32(existingHook.ID), localVarPostBody, []string{"application/json"})
		if err != nil {
			return errors.Errorf("failed to update webhook. err: %+v", err)
		}

		logger.With("hook-id", existingHook.ID).Info("hook successfully updated")
	}

	return nil
}
