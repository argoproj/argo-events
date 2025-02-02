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

package bitbucket

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	bitbucketv2 "github.com/ktrysmt/go-bitbucket"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
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
// 4. PostInactivate

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

	request.Body = http.MaxBytesReader(writer, request.Body, route.Context.GetMaxPayloadSize())
	body, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Desugar().Error("failed to parse request body", zap.Error(err))
		sharedutil.SendErrorResponse(writer, err.Error())
		return
	}

	event := &events.BitbucketEventData{
		Headers:  request.Header,
		Body:     (*json.RawMessage)(&body),
		Metadata: router.bitbucketEventSource.Metadata,
	}

	eventBody, err := json.Marshal(event)
	if err != nil {
		logger.Info("failed to marshal event")
		sharedutil.SendErrorResponse(writer, "invalid event")
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
	bitbucketEventSource := router.bitbucketEventSource
	logger := router.GetRoute().Logger

	if bitbucketEventSource.DeleteHookOnFinish && len(router.hookIDs) > 0 {
		logger.Info("deleting webhooks from bitbucket...")

		for _, repo := range bitbucketEventSource.GetBitbucketRepositories() {
			hookID, ok := router.hookIDs[repo.GetRepositoryID()]
			if !ok {
				return fmt.Errorf("can not find hook ID for repo key: %s", repo.GetRepositoryID())
			}

			if err := router.deleteWebhook(repo, hookID); err != nil {
				logger.Errorw("failed to delete webhook",
					zap.String("owner", repo.Owner), zap.String("repository-slug", repo.RepositorySlug), zap.Error(err))
				return fmt.Errorf("failed to delete hook for repo %s/%s, %w", repo.Owner, repo.RepositorySlug, err)
			}

			logger.Info("successfully deleted hook for repo",
				zap.String("owner", repo.Owner), zap.String("repository-slug", repo.RepositorySlug))
		}
	}

	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	defer sources.Recover(el.GetEventName())

	bitbucketEventSource := &el.BitbucketEventSource
	logger := logging.FromContext(ctx).With(
		logging.LabelEventSourceType, el.GetEventSourceType(),
		logging.LabelEventName, el.GetEventName(),
	)

	logger.Info("started processing the Bitbucket event source...")
	route := webhook.NewRoute(bitbucketEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	router := &Router{
		route:                route,
		bitbucketEventSource: bitbucketEventSource,
		hookIDs:              make(map[string]string),
	}

	if !bitbucketEventSource.ShouldCreateWebhooks() {
		logger.Info("access token or webhook configuration were not provided, skipping webhooks creation")
		return webhook.ManageRoute(ctx, router, controller, dispatch)
	}

	logger.Info("choosing bitbucket auth strategy...")
	authStrategy, err := router.chooseAuthStrategy()
	if err != nil {
		return fmt.Errorf("failed to get bitbucket auth strategy, %w", err)
	}

	router.client = authStrategy.BitbucketClient()

	applyWebhooks := func() {
		for _, repo := range bitbucketEventSource.GetBitbucketRepositories() {
			if err = router.applyBitbucketWebhook(repo); err != nil {
				logger.Errorw("failed to apply Bitbucket webhook",
					zap.String("owner", repo.Owner), zap.String("repository-slug", repo.RepositorySlug), zap.Error(err))
				continue
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	// When running multiple replicas of the eventsource, they will all try to create the webhook.
	// Randomly sleep some time to mitigate the issue.
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(5000)))
	time.Sleep(time.Duration(randomNum.Int64()) * time.Millisecond)
	applyWebhooks()

	return webhook.ManageRoute(ctx, router, controller, dispatch)
}

// chooseAuthStrategy returns an AuthStrategy based on the given credentials
func (router *Router) chooseAuthStrategy() (AuthStrategy, error) {
	es := router.bitbucketEventSource
	switch {
	case es.HasBitbucketBasicAuth():
		return NewBasicAuthStrategy(es.Auth.Basic.Username, es.Auth.Basic.Password)
	case es.HasBitbucketOAuthToken():
		return NewOAuthTokenAuthStrategy(es.Auth.OAuthToken)
	default:
		return nil, fmt.Errorf("none of the supported auth options were provided")
	}
}

// applyBitbucketWebhook creates or updates the configured webhook in Bitbucket
func (router *Router) applyBitbucketWebhook(repo aev1.BitbucketRepository) error {
	bitbucketEventSource := router.bitbucketEventSource
	route := router.route
	logger := router.GetRoute().Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
		"owner", repo.Owner,
		"repository-slug", repo.RepositorySlug,
	)

	formattedWebhookURL := sharedutil.FormattedURL(bitbucketEventSource.Webhook.URL, bitbucketEventSource.Webhook.Endpoint)

	logger.Info("listing existing webhooks...")
	hooks, err := router.listWebhooks(repo)
	if err != nil {
		logger.Errorw("failed to list webhooks", zap.Error(err))
		return fmt.Errorf("failed to list webhooks, %w", err)
	}

	logger.Info("checking if webhook already exists...")
	existingHookSubscription, isFound := router.findWebhook(hooks, formattedWebhookURL)
	if isFound {
		logger.Info("webhook already exists, removing old webhook...")
		if err := router.deleteWebhook(repo, existingHookSubscription.Uuid); err != nil {
			logger.Errorw("failed to delete old webhook",
				zap.String("owner", repo.Owner), zap.String("repository-slug", repo.RepositorySlug), zap.Error(err))
			return fmt.Errorf("failed to delete old webhook for repo %s/%s, %w", repo.Owner, repo.RepositorySlug, err)
		}
	}

	logger.Info("creating a new webhook...")
	newWebhook, err := router.createWebhook(repo, formattedWebhookURL)
	if err != nil {
		logger.Errorw("failed to create new webhook", zap.Error(err))
		return fmt.Errorf("failed to create new webhook, %w", err)
	}

	router.hookIDs[repo.GetRepositoryID()] = newWebhook.Uuid

	logger.Info("successfully created a new webhook")
	return nil
}

// createWebhook creates a new webhook
func (router *Router) createWebhook(repo aev1.BitbucketRepository, formattedWebhookURL string) (*bitbucketv2.Webhook, error) {
	opt := &bitbucketv2.WebhooksOptions{
		Owner:       repo.Owner,
		RepoSlug:    repo.RepositorySlug,
		Url:         formattedWebhookURL,
		Description: "webhook managed by Argo-Events",
		Active:      true,
		Events:      router.bitbucketEventSource.Events,
	}

	return router.client.Repositories.Webhooks.Create(opt)
}

// deleteWebhook deletes an existing webhook
func (router *Router) deleteWebhook(repo aev1.BitbucketRepository, hookID string) error {
	_, err := router.client.Repositories.Webhooks.Delete(&bitbucketv2.WebhooksOptions{
		Owner:    repo.Owner,
		RepoSlug: repo.RepositorySlug,
		Uuid:     hookID,
	})
	if err != nil {
		// Skip not found errors in case the webhook was already deleted
		var bitbucketErr *bitbucketv2.UnexpectedResponseStatusError
		if errors.As(err, &bitbucketErr) && bitbucketErr.Status == "404 Not Found" {
			return nil
		}
	}

	return err
}

// listWebhooks gets a list of all existing webhooks in target repository
func (router *Router) listWebhooks(repo aev1.BitbucketRepository) ([]WebhookSubscription, error) {
	hooksResponse, err := router.client.Repositories.Webhooks.Gets(&bitbucketv2.WebhooksOptions{
		Owner:    repo.Owner,
		RepoSlug: repo.RepositorySlug,
	})
	if err != nil {
		return nil, err
	}

	return router.extractHooksFromListResponse(hooksResponse)
}

// extractHooksFromListResponse helper that extracts the list of webhooks from the response of listWebhooks
func (router *Router) extractHooksFromListResponse(listHooksResponse interface{}) ([]WebhookSubscription, error) {
	logger := router.GetRoute().Logger
	res, ok := listHooksResponse.(map[string]interface{})
	if !ok {
		logger.Errorw("failed to parse the list webhooks response", zap.Any("response", listHooksResponse))
		return nil, fmt.Errorf("failed to parse the list webhooks response")
	}

	var hooks []WebhookSubscription
	err := mapstructure.Decode(res["values"], &hooks)
	if err != nil || hooks == nil {
		logger.Errorw("failed to parse the list webhooks response", zap.Any("response", listHooksResponse))
		return nil, fmt.Errorf("failed to parse the list webhooks response")
	}

	return hooks, nil
}

// findWebhook searches for a webhook in a list by its URL and returns the webhook if its found
func (router *Router) findWebhook(hooks []WebhookSubscription, targetWebhookURL string) (*WebhookSubscription, bool) {
	var existingHookSubscription *WebhookSubscription
	isFound := false
	for _, hook := range hooks {
		if hook.Url == targetWebhookURL {
			isFound = true
			existingHookSubscription = &hook
			break
		}
	}

	return existingHookSubscription, isFound
}
