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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"net/http"
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

	body, err := router.parseAndValidateBitbucketServerRequest(request)
	if err != nil {
		logger.Errorw("failed to parse/validate request", zap.Error(err))
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
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
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

	if bitbucketserverEventSource.WebhookSecret != nil {
		logger.Info("retrieving the webhook secret...")
		webhookSecret, err := common.GetSecretFromVolume(bitbucketserverEventSource.WebhookSecret)
		if err != nil {
			return errors.Errorf("failed to get bitbucketserver webhook secret. err: %+v", err)
		}

		router.hookSecret = webhookSecret
	}

	logger.Info("setting up the client to connect to Bitbucket Server...")
	bitbucketConfig := bitbucketv1.NewConfiguration(bitbucketserverEventSource.BitbucketServerBaseURL)
	bitbucketConfig.AddDefaultHeader("x-atlassian-token", "no-check")
	bitbucketConfig.AddDefaultHeader("x-requested-with", "XMLHttpRequest")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = context.WithValue(ctx, bitbucketv1.ContextAccessToken, bitbucketToken)

	applyWebhooks := func() {
		for _, repo := range bitbucketserverEventSource.GetBitbucketServerRepositories() {
			if err = router.applyBitbucketServerWebhook(ctx, bitbucketConfig, repo); err != nil {
				logger.Errorw("failed to create/update Bitbucket webhook",
					zap.String("project-key", repo.ProjectKey), zap.String("repository-slug", repo.RepositorySlug), zap.Error(err))
				continue
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	// When running multiple replicas of the eventsource, they will all try to create the webhook.
	// Randomly sleep some time to mitigate the issue.
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(2000)))
	time.Sleep(time.Duration(randomNum.Int64()) * time.Millisecond)
	applyWebhooks()

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
				applyWebhooks()
			}
		}
	}()

	return webhook.ManageRoute(ctx, router, controller, dispatch)
}

// applyBitbucketServerWebhook creates or updates the configured webhook in Bitbucket
func (router *Router) applyBitbucketServerWebhook(ctx context.Context, bitbucketConfig *bitbucketv1.Configuration, repo v1alpha1.BitbucketServerRepository) error {
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

	hooks, err := router.listWebhooks(bitbucketClient, repo)
	if err != nil {
		return errors.Wrapf(err, "failed to list existing hooks to check for duplicates for repository %s/%s", repo.ProjectKey, repo.RepositorySlug)
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

	newHook := bitbucketv1.Webhook{
		Name:          "Argo Events",
		Url:           formattedURL,
		Active:        true,
		Events:        bitbucketserverEventSource.Events,
		Configuration: bitbucketv1.WebhookConfiguration{Secret: router.hookSecret},
	}

	requestBody, err := router.createRequestBodyFromWebhook(newHook)
	if err != nil {
		return errors.Wrapf(err, "failed to create request body from webhook")
	}

	// Update the webhook when it does exist and the events/configuration have changed
	if isAlreadyExists {
		logger.Info("webhook already exists")
		if router.shouldUpdateWebhook(existingHook, newHook) {
			logger.Info("webhook requires an update")
			err = router.updateWebhook(bitbucketClient, existingHook.ID, requestBody, repo)
			if err != nil {
				return errors.Errorf("failed to update webhook. err: %+v", err)
			}

			logger.With("hook-id", existingHook.ID).Info("hook successfully updated")
		}

		return nil
	}

	// Create the webhook when it doesn't exist yet
	createdHook, err := router.createWebhook(bitbucketClient, requestBody, repo)
	if err != nil {
		return errors.Errorf("failed to create webhook. err: %+v", err)
	}

	router.hookIDs[repo.ProjectKey+","+repo.RepositorySlug] = createdHook.ID

	logger.With("hook-id", createdHook.ID).Info("hook successfully registered")

	return nil
}

func (router *Router) listWebhooks(bitbucketClient *bitbucketv1.APIClient, repo v1alpha1.BitbucketServerRepository) ([]bitbucketv1.Webhook, error) {
	apiResponse, err := bitbucketClient.DefaultApi.FindWebhooks(repo.ProjectKey, repo.RepositorySlug, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list existing hooks to check for duplicates for repository %s/%s", repo.ProjectKey, repo.RepositorySlug)
	}

	hooks, err := bitbucketv1.GetWebhooksResponse(apiResponse)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert the list of webhooks for repository %s/%s", repo.ProjectKey, repo.RepositorySlug)
	}

	return hooks, nil
}

func (router *Router) createWebhook(bitbucketClient *bitbucketv1.APIClient, requestBody []byte, repo v1alpha1.BitbucketServerRepository) (*bitbucketv1.Webhook, error) {
	apiResponse, err := bitbucketClient.DefaultApi.CreateWebhook(repo.ProjectKey, repo.RepositorySlug, requestBody, []string{"application/json"})
	if err != nil {
		return nil, errors.Errorf("failed to add webhook. err: %+v", err)
	}

	var createdHook *bitbucketv1.Webhook
	err = mapstructure.Decode(apiResponse.Values, &createdHook)
	if err != nil {
		return nil, errors.Errorf("failed to convert API response to Webhook struct. err: %+v", err)
	}

	return createdHook, nil
}

func (router *Router) updateWebhook(bitbucketClient *bitbucketv1.APIClient, hookID int, requestBody []byte, repo v1alpha1.BitbucketServerRepository) error {
	_, err := bitbucketClient.DefaultApi.UpdateWebhook(repo.ProjectKey, repo.RepositorySlug, int32(hookID), requestBody, []string{"application/json"})

	return err
}

func (router *Router) shouldUpdateWebhook(existingHook bitbucketv1.Webhook, newHook bitbucketv1.Webhook) bool {
	return !common.ElementsMatch(existingHook.Events, newHook.Events) ||
		existingHook.Configuration.Secret != newHook.Configuration.Secret
}

func (router *Router) createRequestBodyFromWebhook(hook bitbucketv1.Webhook) ([]byte, error) {
	var err error
	var finalHook interface{} = hook

	// if the hook doesn't have a secret, the configuration field must be removed in order for the request to succeed,
	// otherwise Bitbucket Server sends 500 response because of empty string value in the hook.Configuration.Secret field
	if hook.Configuration.Secret == "" {
		hookMap := make(map[string]interface{})
		err = common.StructToMap(hook, hookMap)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert webhook to map")
		}

		delete(hookMap, "configuration")

		finalHook = hookMap
	}

	requestBody, err := json.Marshal(finalHook)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal new webhook to JSON")
	}

	return requestBody, nil
}

func (router *Router) parseAndValidateBitbucketServerRequest(request *http.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse request body")
	}

	if len(router.hookSecret) != 0 {
		signature := request.Header.Get("X-Hub-Signature")
		if len(signature) == 0 {
			return nil, errors.New("missing signature header")
		}

		mac := hmac.New(sha256.New, []byte(router.hookSecret))
		_, _ = mac.Write(body)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature[7:]), []byte(expectedMAC)) {
			return nil, errors.New("hmac verification failed")
		}
	}

	return body, nil
}
