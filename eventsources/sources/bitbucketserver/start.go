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
	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/pkg/errors"
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

// getCredentials retrieves credentials to connect to Bitbucket Server
func (router *Router) getCredentials(keySelector *corev1.SecretKeySelector) (*cred, error) {
	token, err := common.GetSecretFromVolume(keySelector)
	if err != nil {
		return nil, errors.Wrap(err, "token not founnd")
	}
	return &cred{
		token: token,
	}, nil
}

// getCredentials retrieves credentials to connect to Bitbucket Server
func (router *Router) getWebhookSecret(keySelector *corev1.SecretKeySelector) (*webhookSecret, error) {
	secret, err := common.GetSecretFromVolume(keySelector)
	if err != nil {
		return nil, errors.Wrap(err, "token not founnd")
	}
	return &webhookSecret{
		secret: secret,
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

	logger := route.Logger.With(
		"project-key", bitbucketserverEventSource.ProjectKey,
		"repository-slug", bitbucketserverEventSource.RepositorySlug,
		"hook-id", router.hookID,
	)

	if bitbucketserverEventSource.DeleteHookOnFinish && router.hookID > 0 {
		logger.Info("deleting webhook from bitbucket")

		bitbucketCredentials, err := router.getCredentials(bitbucketserverEventSource.AccessToken)
		if err != nil {
			return errors.Errorf("failed to get bitbucketserver credentials. err: %+v", err)
		}

		bitbucketConfig := bitbucketv1.NewConfiguration(bitbucketserverEventSource.BitbucketServerBaseURL)
		bitbucketConfig.AddDefaultHeader("x-atlassian-token", "no-check")
		bitbucketConfig.AddDefaultHeader("x-requested-with", "XMLHttpRequest")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ctx = context.WithValue(ctx, bitbucketv1.ContextAccessToken, bitbucketCredentials.token)

		bitbucketClient := bitbucketv1.NewAPIClient(ctx, bitbucketConfig)

		_, err = bitbucketClient.DefaultApi.DeleteWebhook(bitbucketserverEventSource.ProjectKey, bitbucketserverEventSource.RepositorySlug, int32(router.hookID))
		if err != nil {
			return errors.Errorf("failed to delete bitbucketserver webhook. err: %+v", err)
		}

		logger.Info("bitbucket server webhook deleted")
	} else {
		logger.Info("hook not found, not deleting")
	}

	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	defer sources.Recover(el.GetEventName())

	bitbucketserverEventSource := &el.BitbucketServerEventSource

	logger := logging.FromContext(ctx).With(
		logging.LabelEventSourceType, el.GetEventSourceType(),
		logging.LabelEventName, el.GetEventName(),
		"project-key", bitbucketserverEventSource.ProjectKey,
		"repository-slug", bitbucketserverEventSource.RepositorySlug,
		"base-url", bitbucketserverEventSource.BitbucketServerBaseURL,
	)

	logger.Info("started processing the Bitbucket Server event source...")

	route := webhook.NewRoute(bitbucketserverEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	router := &Router{
		route:                      route,
		bitbucketserverEventSource: bitbucketserverEventSource,
	}

	logger.Info("retrieving the access token credentials...")
	bitbucketCredentials, err := router.getCredentials(bitbucketserverEventSource.AccessToken)
	if err != nil {
		return errors.Errorf("failed to get bitbucketserver credentials. err: %+v", err)
	}

	logger.Info("setting up the client to connect to Bitbucket Server...")
	bitbucketConfig := bitbucketv1.NewConfiguration(bitbucketserverEventSource.BitbucketServerBaseURL)
	bitbucketConfig.AddDefaultHeader("x-atlassian-token", "no-check")
	bitbucketConfig.AddDefaultHeader("x-requested-with", "XMLHttpRequest")

	// When running multiple replicas of the eventsource, they will all try to create the webhook.
	// Randomly sleep some time to mitigate the issue.
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	time.Sleep(time.Duration(r1.Intn(2000)) * time.Millisecond)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = context.WithValue(ctx, bitbucketv1.ContextAccessToken, bitbucketCredentials.token)
	err = router.CreateBitbucketWebhook(ctx, bitbucketConfig)

	if err != nil {
		logger.Errorw("failed to create/update Bitbucket webhook", zap.Error(err))
	}

	go func() {
		// Another kind of race conditions might happen when pods do rolling upgrade - new pod starts
		// and old pod terminates, if DeleteHookOnFinish is true, the hook will be deleted from Bitbucket.
		// This is a workround to mitigate the race conditions.
		logger.Info("starting bitbucket hooks manager daemon")
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("exiting bitbucket hooks manager daemon")
				return
			case <-ticker.C:
				err := router.CreateBitbucketWebhook(ctx, bitbucketConfig)

				if err != nil {
					logger.Errorw("failed to create/update Bitbucket webhook", zap.Error(err))
				}
			}
		}
	}()

	return webhook.ManageRoute(ctx, router, controller, dispatch)
}

func (router *Router) CreateBitbucketWebhook(ctx context.Context, bitbucketConfig *bitbucketv1.Configuration) error {
	bitbucketserverEventSource := router.bitbucketserverEventSource
	route := router.route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
		"project-key", bitbucketserverEventSource.ProjectKey,
		"repository-slug", bitbucketserverEventSource.RepositorySlug,
		"base-url", bitbucketserverEventSource.BitbucketServerBaseURL,
	)

	bitbucketClient := bitbucketv1.NewAPIClient(ctx, bitbucketConfig)

	formattedURL := common.FormattedURL(bitbucketserverEventSource.Webhook.URL, bitbucketserverEventSource.Webhook.Endpoint)

	apiResponse, err := bitbucketClient.DefaultApi.FindWebhooks(bitbucketserverEventSource.ProjectKey, bitbucketserverEventSource.RepositorySlug, nil)

	if err != nil {
		return errors.Wrapf(err, "failed to list existing hooks to check for duplicates for repository %s/%s", router.bitbucketserverEventSource.ProjectKey, router.bitbucketserverEventSource.RepositorySlug)
	}

	hooks, err := bitbucketv1.GetWebhooksResponse(apiResponse)

	if err != nil {
		return errors.Wrapf(err, "failed to convert the list of webhooks for repository %s/%s", router.bitbucketserverEventSource.ProjectKey, router.bitbucketserverEventSource.RepositorySlug)
	}

	var existingHook bitbucketv1.Webhook
	isAlreadyExists := false

	for _, hook := range hooks {
		if hook.Url == formattedURL {
			isAlreadyExists = true
			existingHook = hook
			router.hookID = hook.ID
			break
		}
	}

	logger.Info("retrieving the webhook secret...")
	webhookSecret, err := router.getWebhookSecret(bitbucketserverEventSource.WebhookSecret)
	if err != nil {
		return errors.Errorf("failed to get bitbucketserver webhook secret. err: %+v", err)
	}

	newHook := bitbucketv1.Webhook{
		Name:          "Argo Events",
		Url:           formattedURL,
		Active:        true,
		Events:        bitbucketserverEventSource.Events,
		Configuration: bitbucketv1.WebhookConfiguration{Secret: webhookSecret.secret},
	}

	localVarPostBody, err := json.Marshal(newHook)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal new webhook to JSON")
	}

	var createdHook *bitbucketv1.Webhook

	// Create the webhook when it doesn't exist yet
	if !isAlreadyExists {
		apiResponse, err = bitbucketClient.DefaultApi.CreateWebhook(bitbucketserverEventSource.ProjectKey, bitbucketserverEventSource.RepositorySlug, localVarPostBody, []string{"application/json"})

		if err != nil {
			return errors.Errorf("failed to add webhook. err: %+v", err)
		}

		err = mapstructure.Decode(apiResponse.Values, &createdHook)
		if err != nil {
			return errors.Errorf("failed to convert API response to Webhook struct. err: %+v", err)
		}
		router.hookID = createdHook.ID

		logger.With("hook-id", createdHook.ID).Info("hook succesfully registered")
	}

	// Update the webhook when it does exist and the configuration has chagned
	if isAlreadyExists && (!reflect.DeepEqual(existingHook.Events, newHook.Events) || !reflect.DeepEqual(existingHook.Configuration, newHook.Configuration)) {
		logger.Info("webhook already exists and configuration has changed. Updating webhook.")

		apiResponse, err = bitbucketClient.DefaultApi.UpdateWebhook(bitbucketserverEventSource.ProjectKey, bitbucketserverEventSource.RepositorySlug, int32(existingHook.ID), localVarPostBody, []string{"application/json"})

		if err != nil {
			return errors.Errorf("failed to update webhook. err: %+v", err)
		}

		err = mapstructure.Decode(apiResponse.Values, &createdHook)
		if err != nil {
			return errors.Errorf("failed to convert API response to Webhook struct. err: %+v", err)
		}

		logger.With("hook-id", createdHook.ID).Info("hook succesfully updated")
	}

	return nil
}
