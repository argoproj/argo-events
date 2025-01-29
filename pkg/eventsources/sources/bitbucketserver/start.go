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
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// controller controls the webhook operations
var controller = webhook.NewController()

// set up the activation and inactivation channels to control the state of routes.
func init() {
	go webhook.ProcessRouteStatus(controller)
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	defer sources.Recover(el.GetEventName())

	bitbucketServerEventSource := &el.BitbucketServerEventSource

	logger := logging.FromContext(ctx).With(
		logging.LabelEventSourceType, el.GetEventSourceType(),
		logging.LabelEventName, el.GetEventName(),
		"base-url", bitbucketServerEventSource.BitbucketServerBaseURL,
	)

	logger.Info("started processing the bitbucketserver event source...")

	route := webhook.NewRoute(bitbucketServerEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	router := &Router{
		route:                      route,
		bitbucketServerEventSource: bitbucketServerEventSource,
		hookIDs:                    make(map[string]int),
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if bitbucketServerEventSource.AccessToken != nil {
		err := setupBitbucketServerClients(ctx, router, bitbucketServerEventSource)
		if err != nil {
			return fmt.Errorf("setting up bitbucketserver clients failed: %w", err)
		}
	} else {
		logger.Info("skipping setting up bitbucketserver clients: access token not specified")
	}

	if bitbucketServerEventSource.ShouldCreateWebhooks() {
		err := router.manageBitbucketServerWebhooks(ctx, bitbucketServerEventSource)
		if err != nil {
			return fmt.Errorf("managing bitbucketserver webhooks failed: %w", err)
		}
	} else {
		logger.Info("skipping managing bitbucketserver webhooks: access token or webhook url or projects or repositories not specified")
	}

	return webhook.ManageRoute(ctx, router, controller, dispatch)
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
	bitbucketServerEventSource := router.bitbucketServerEventSource
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

	bitbucketServerEvents, err := processBitbucketServerWebhook(
		router.customClient,
		router.hookSecret,
		writer,
		request,
		route.Context.GetMaxPayloadSize(),
		bitbucketServerEventSource.SkipBranchRefsChangedOnOpenPR,
		bitbucketServerEventSource.OneEventPerChange,
	)
	if err != nil {
		logger.Errorw("failed to handle event", zap.Error(err))
		sharedutil.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	for _, bitbucketServerEvent := range bitbucketServerEvents {
		event := &events.BitbucketServerEventData{
			Headers:  request.Header,
			Body:     (*json.RawMessage)(&bitbucketServerEvent),
			Metadata: router.bitbucketServerEventSource.Metadata,
		}

		eventBody, jsonErr := json.Marshal(event)
		if jsonErr != nil {
			logger.Errorw("failed to parse event", zap.Error(jsonErr))
			sharedutil.SendErrorResponse(writer, "invalid event")
			route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
			return
		}

		webhook.DispatchEvent(route, eventBody, logger, writer)
	}

	logger.Info("request successfully processed")
	sharedutil.SendSuccessResponse(writer, "success")
}

// PostActivate performs operations once the route is activated and ready to consume requests
func (router *Router) PostActivate() error {
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (router *Router) PostInactivate() error {
	bitbucketServerEventSource := router.bitbucketServerEventSource

	if !bitbucketServerEventSource.DeleteHookOnFinish || len(router.hookIDs) == 0 {
		return nil
	}

	return router.deleteWebhooks(bitbucketServerEventSource)
}

func (router *Router) listWebhooks(repo v1alpha1.BitbucketServerRepository) ([]bitbucketv1.Webhook, error) {
	apiResponse, err := router.client.DefaultApi.FindWebhooks(repo.ProjectKey, repo.RepositorySlug, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing hooks to check for duplicates for repository %s/%s: %w", repo.ProjectKey, repo.RepositorySlug, err)
	}

	hooks, err := bitbucketv1.GetWebhooksResponse(apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the list of webhooks for repository %s/%s: %w", repo.ProjectKey, repo.RepositorySlug, err)
	}

	return hooks, nil
}

func (router *Router) createWebhook(requestBody []byte, repo v1alpha1.BitbucketServerRepository) (*bitbucketv1.Webhook, error) {
	apiResponse, err := router.client.DefaultApi.CreateWebhook(repo.ProjectKey, repo.RepositorySlug, requestBody, []string{"application/json"})
	if err != nil {
		return nil, fmt.Errorf("failed to add webhook: %w", err)
	}

	var createdHook *bitbucketv1.Webhook
	err = mapstructure.Decode(apiResponse.Values, &createdHook)
	if err != nil {
		return nil, fmt.Errorf("failed to convert API response to webhook struct: %w", err)
	}

	return createdHook, nil
}

func (router *Router) updateWebhook(hookID int, requestBody []byte, repo v1alpha1.BitbucketServerRepository) error {
	_, err := router.client.DefaultApi.UpdateWebhook(repo.ProjectKey, repo.RepositorySlug, int32(hookID), requestBody, []string{"application/json"})

	return err
}

func (router *Router) shouldUpdateWebhook(existingHook bitbucketv1.Webhook, newHook bitbucketv1.Webhook) bool {
	return !sharedutil.ElementsMatch(existingHook.Events, newHook.Events) ||
		existingHook.Configuration.Secret != newHook.Configuration.Secret
}

func (router *Router) createRequestBodyFromWebhook(hook bitbucketv1.Webhook) ([]byte, error) {
	var err error
	var finalHook interface{} = hook

	// if the hook doesn't have a secret, the configuration field must be removed in order for the request to succeed,
	// otherwise Bitbucket Server sends 500 response because of empty string value in the hook.Configuration.Secret field
	if hook.Configuration.Secret == "" {
		hookMap := make(map[string]interface{})
		err = sharedutil.StructToMap(hook, hookMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert webhook to map: %w", err)
		}

		delete(hookMap, "configuration")

		finalHook = hookMap
	}

	requestBody, err := json.Marshal(finalHook)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new webhook to JSON: %w", err)
	}

	return requestBody, nil
}

func (router *Router) manageBitbucketServerWebhooks(ctx context.Context, bitbucketServerEventSource *v1alpha1.BitbucketServerEventSource) error {
	if bitbucketServerEventSource.WebhookSecret != nil {
		webhookSecret, err := sharedutil.GetSecretFromVolume(bitbucketServerEventSource.WebhookSecret)
		if err != nil {
			return fmt.Errorf("getting bitbucketserver webhook secret: %w", err)
		}

		router.hookSecret = webhookSecret
	}

	// When running multiple replicas of this event source, they will try to create webhooks at the same time.
	// Randomly delay running the initial apply webhooks func to mitigate the issue.
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(2000)))
	time.Sleep(time.Duration(randomNum.Int64()) * time.Millisecond)

	// Initial apply of webhooks on event source start.
	err := router.applyBitbucketServerWebhooks(bitbucketServerEventSource)
	if err != nil {
		return fmt.Errorf("initial applying bitbucketserver webhooks failed: %w", err)
	}

	var checkInterval time.Duration
	if bitbucketServerEventSource.CheckInterval == "" {
		checkInterval = 60 * time.Second
	} else {
		checkInterval, err = time.ParseDuration(bitbucketServerEventSource.CheckInterval)
		if err != nil {
			return err
		}
	}

	go func() {
		// Another kind of race conditions might happen when pods do rolling upgrade - new pod starts
		// and old pod terminates, if DeleteHookOnFinish is true, the hook will be deleted from Bitbucket.
		// This is a workaround to mitigate the race conditions.
		router.route.Logger.Info("starting bitbucketserver webhooks manager")

		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				router.route.Logger.Info("exiting bitbucketserver webhooks manager")
				return
			case <-ticker.C:
				err = router.applyBitbucketServerWebhooks(bitbucketServerEventSource)
				if err != nil {
					router.route.Logger.Errorf("re-applying bitbucketserver webhooks failed: %w", err)
				}
			}
		}
	}()

	return nil
}

// applyBitbucketServerWebhooks ensures all configured repositories have the correct webhook settings applied.
func (router *Router) applyBitbucketServerWebhooks(bitbucketServerEventSource *v1alpha1.BitbucketServerEventSource) error {
	bitbucketRepositories := bitbucketServerEventSource.GetBitbucketServerRepositories()

	if len(bitbucketServerEventSource.Projects) > 0 {
		bitbucketProjectRepositories, err := getProjectRepositories(router.client, bitbucketServerEventSource.Projects)
		if err != nil {
			return fmt.Errorf("getting list of bitbucketserver repositories from projects %v: %w", bitbucketServerEventSource.Projects, err)
		}

		bitbucketRepositories = append(bitbucketRepositories, bitbucketProjectRepositories...)
	}

	for _, repo := range bitbucketRepositories {
		if err := router.applyBitbucketServerWebhook(repo); err != nil {
			router.route.Logger.Errorw("failed to apply bitbucketserver webhook",
				zap.String("project-key", repo.ProjectKey), zap.String("repository-slug", repo.RepositorySlug), zap.Error(err))
			continue
		}

		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

// applyBitbucketServerWebhook creates or updates the configured webhook in Bitbucket Server.
func (router *Router) applyBitbucketServerWebhook(repo v1alpha1.BitbucketServerRepository) error {
	bitbucketServerEventSource := router.bitbucketServerEventSource
	route := router.route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
		"project-key", repo.ProjectKey,
		"repository-slug", repo.RepositorySlug,
	)

	formattedURL := sharedutil.FormattedURL(bitbucketServerEventSource.Webhook.URL, bitbucketServerEventSource.Webhook.Endpoint)

	hooks, err := router.listWebhooks(repo)
	if err != nil {
		return fmt.Errorf("failed to list existing hooks to check for duplicates for repository %s/%s: %w", repo.ProjectKey, repo.RepositorySlug, err)
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
		Events:        bitbucketServerEventSource.Events,
		Configuration: bitbucketv1.WebhookConfiguration{Secret: router.hookSecret},
	}

	requestBody, err := router.createRequestBodyFromWebhook(newHook)
	if err != nil {
		return fmt.Errorf("failed to create request body from webhook: %w", err)
	}

	// Update the webhook when it does exist and the events/configuration have changed.
	if isAlreadyExists {
		if router.shouldUpdateWebhook(existingHook, newHook) {
			err = router.updateWebhook(existingHook.ID, requestBody, repo)
			if err != nil {
				return fmt.Errorf("failed to update bitbucketserver webhook: %w", err)
			}

			logger.Infow("bitbucketserver webhook successfully updated", zap.Int("webhook-id", existingHook.ID))
		}

		return nil
	}

	// Create the webhook when it doesn't exist yet.
	createdHook, err := router.createWebhook(requestBody, repo)
	if err != nil {
		return fmt.Errorf("failed to create bitbucketserver webhook: %w", err)
	}

	router.hookIDs[repo.ProjectKey+","+repo.RepositorySlug] = createdHook.ID

	logger.Infow("bitbucketserver webhook successfully registered", zap.Int("webhook-id", createdHook.ID))

	return nil
}

func (router *Router) deleteWebhooks(bitbucketServerEventSource *v1alpha1.BitbucketServerEventSource) error {
	bitbucketRepositories := bitbucketServerEventSource.GetBitbucketServerRepositories()

	if len(bitbucketServerEventSource.Projects) > 0 {
		bitbucketProjectRepositories, err := getProjectRepositories(router.deleteClient, bitbucketServerEventSource.Projects)
		if err != nil {
			return fmt.Errorf("getting list of bitbucketserver repositories from projects %v: %w", bitbucketServerEventSource.Projects, err)
		}

		bitbucketRepositories = append(bitbucketRepositories, bitbucketProjectRepositories...)
	}

	for _, repo := range bitbucketRepositories {
		id, ok := router.hookIDs[repo.ProjectKey+","+repo.RepositorySlug]
		if !ok {
			router.route.Logger.Warnw("webhook id for bitbucketserver repository not defined",
				zap.String("project-key", repo.ProjectKey),
				zap.String("repository-slug", repo.RepositorySlug),
			)
			continue
		}

		_, err := router.deleteClient.DefaultApi.DeleteWebhook(repo.ProjectKey, repo.RepositorySlug, int32(id))
		if err != nil {
			return fmt.Errorf("failed to delete bitbucketserver webhook: %w", err)
		}

		router.route.Logger.Infow("bitbucketserver webhook deleted",
			zap.Int("webhook-id", id),
			zap.String("project-key", repo.ProjectKey),
			zap.String("repository-slug", repo.RepositorySlug),
		)
	}

	return nil
}

func setupBitbucketServerClients(ctx context.Context, router *Router, bitbucketServerEventSource *v1alpha1.BitbucketServerEventSource) error {
	bitbucketToken, err := sharedutil.GetSecretFromVolume(bitbucketServerEventSource.AccessToken)
	if err != nil {
		return fmt.Errorf("retrieving the bitbucketserver access token credentials: %w", err)
	}

	bitbucketConfig, err := newBitbucketServerClientCfg(bitbucketServerEventSource)
	if err != nil {
		return fmt.Errorf("initializing bitbucketserver client config: %w", err)
	}

	bitbucketURL, err := url.Parse(bitbucketServerEventSource.BitbucketServerBaseURL)
	if err != nil {
		return fmt.Errorf("parsing bitbucketserver url: %w", err)
	}

	router.client = newBitbucketServerClient(ctx, bitbucketConfig, bitbucketToken)
	router.deleteClient = newBitbucketServerClient(context.Background(), bitbucketConfig, bitbucketToken)
	router.customClient = &customBitbucketServerClient{
		client: bitbucketConfig.HTTPClient,
		ctx:    ctx,
		token:  bitbucketToken,
		url:    bitbucketURL,
	}

	return nil
}

// processBitbucketServerWebhook processes Bitbucket Server webhook events specifically looking for "repo:refs_changed" events.
// It first checks if the incoming webhook is configured to be handled (i.e., it's a "repo:refs_changed" event).
// If not, it logs the discrepancy and returns the raw event.
//
// For valid "repo:refs_changed" events, it:
//   - Unmarshals the JSON body into a refsChangedWebhookEvent struct.
//   - Optionally filters out changes associated with open pull requests if SkipBranchRefsChangedOnOpenPR is enabled.
//   - Depending on the OneEventPerChange setting, it either batches all changes into one event or creates individual events for each change.
//
// It returns a slice of byte slices containing the processed or original webhook events, and any error encountered during processing.
func processBitbucketServerWebhook(customBitbucketServerClient *customBitbucketServerClient, hookSecret string, writer http.ResponseWriter, request *http.Request, maxPayloadSize int64, skipBranchRefsChangedOnOpenPR bool, oneEventPerChange bool) ([][]byte, error) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxPayloadSize)
	body, err := validateBitbucketServerRequest(hookSecret, request)
	if err != nil {
		return nil, fmt.Errorf("failed to validate bitbucketserver webhook request: %w", err)
	}

	if request.Header.Get("X-Event-Key") != "repo:refs_changed" {
		// Don't continue if this is not a repo:refs_changed webhook event.
		return [][]byte{body}, nil
	}

	refsChanged := refsChangedWebhookEvent{}
	err = json.Unmarshal(body, &refsChanged)
	if err != nil {
		return nil, fmt.Errorf("decoding repo:refs_changed webhook: %w", err)
	}

	// When SkipBranchRefsChangedOnOpenPR is enabled, skip publishing the repo:refs_changed event if a Pull Request is already open for the commit.
	// This prevents duplicate notifications when a branch is updated and a PR is already open for the latest commit.
	if skipBranchRefsChangedOnOpenPR {
		err = filterChangesForOpenPRs(customBitbucketServerClient, &refsChanged)
		if err != nil {
			return nil, fmt.Errorf("filtering changes for open prs: %w", err)
		}

		if len(refsChanged.Changes) == 0 {
			// No changes are present in the refsChanged event after filtering. Skip processing event
			return [][]byte{}, nil
		}
	}

	var bitbucketEvents [][]byte

	// Handle event batching based on OneEventPerChange configuration.
	// If enabled, split the refsChanged event into individual events, one per change.
	// Otherwise, send the entire refsChanged event as a single event.
	if oneEventPerChange {
		for _, change := range refsChanged.Changes {
			rc := refsChanged

			rc.Changes = []refsChangedWebHookEventChange{change}

			rcBytes, jsonErr := json.Marshal(&rc)
			if jsonErr != nil {
				return nil, fmt.Errorf("encoding repo:refs_changed webhook: %w", jsonErr)
			}

			bitbucketEvents = append(bitbucketEvents, rcBytes)
		}
	} else {
		bitbucketEvents = append(bitbucketEvents, body)
	}

	return bitbucketEvents, nil
}

// filterChangesForOpenPRs removes changes from the refsChanged webhook event that are linked to open pull requests.
// It examines each change to determine if it pertains to a "BRANCH" that has not been deleted.
// For each relevant change, it checks whether there is an open pull request using the commit's hash.
// Changes with open PRs are excluded from the final list. The modified list of changes is then reassigned
// back to refsChanged.Changes.
func filterChangesForOpenPRs(customBitbucketServerClient *customBitbucketServerClient, refsChanged *refsChangedWebhookEvent) error {
	var keptChanges []refsChangedWebHookEventChange

	for _, change := range refsChanged.Changes {
		c := change

		if c.Ref.Type != "BRANCH" || c.Type == "DELETE" {
			keptChanges = append(keptChanges, c)
			continue
		}

		// Check if the current commit is associated with an open PR.
		hasOpenPR, hasOpenPrErr := refsChangedHasOpenPullRequest(
			customBitbucketServerClient,
			refsChanged.Repository.Project.Key,
			refsChanged.Repository.Slug,
			c.ToHash, // Check for the current change's commit hash
		)
		if hasOpenPrErr != nil {
			return fmt.Errorf("checking if changed branch ref has an open pull request: %w", hasOpenPrErr)
		}

		if !hasOpenPR {
			keptChanges = append(keptChanges, c)
		}
	}

	refsChanged.Changes = keptChanges

	return nil
}

// refsChangedHasOpenPullRequest returns true if the changed commit has an open pull request.
func refsChangedHasOpenPullRequest(customBitbucketServerClient *customBitbucketServerClient, project, repository, commit string) (bool, error) {
	bitbucketPullRequests, err := customBitbucketServerClient.GetCommitPullRequests(project, repository, commit)
	if err != nil {
		return false, fmt.Errorf("getting commit pull requests for project %s, repository %s and commit %s: %w",
			project, repository, commit, err)
	}

	for _, bitbucketPullRequest := range bitbucketPullRequests {
		if strings.EqualFold(bitbucketPullRequest.State, "OPEN") {
			return true, nil
		}
	}

	return false, nil
}

func validateBitbucketServerRequest(hookSecret string, request *http.Request) ([]byte, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	if len(hookSecret) != 0 {
		signature := request.Header.Get("X-Hub-Signature")
		if len(signature) == 0 {
			return nil, fmt.Errorf("missing signature header")
		}

		mac := hmac.New(sha256.New, []byte(hookSecret))
		_, _ = mac.Write(body)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature[7:]), []byte(expectedMAC)) {
			return nil, fmt.Errorf("hmac verification failed")
		}
	}

	return body, nil
}
