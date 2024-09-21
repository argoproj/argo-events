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
	"strconv"
	"strings"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/slices"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/eventsources/events"
	"github.com/argoproj/argo-events/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
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
	bitbucketserverEventSource := router.bitbucketserverEventSource
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

	request.Body = http.MaxBytesReader(writer, request.Body, route.Context.GetMaxPayloadSize())
	body, err := router.parseAndValidateBitbucketServerRequest(request)
	if err != nil {
		logger.Errorw("failed to parse/validate request", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	// When SkipBranchRefsChangedOnOpenPR is enabled and webhook event type is repo:refs_changed,
	// check if a Pull Request is opened for the commit, if one is opened the event will be skipped.
	if bitbucketserverEventSource.SkipBranchRefsChangedOnOpenPR && slices.Contains(bitbucketserverEventSource.Events, "repo:refs_changed") {
		refsChanged := refsChangedWebhookEvent{}
		err := json.Unmarshal(body, &refsChanged)
		if err != nil {
			logger.Errorf("reading webhook body", zap.Error(err))
			common.SendErrorResponse(writer, err.Error())
			route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
			return
		}

		if refsChanged.EventKey == "repo:refs_changed" &&
			len(refsChanged.Changes) > 0 && // Note refsChanged.Changes never has more or less than one change, not sure why Atlassian made it a list.
			strings.EqualFold(refsChanged.Changes[0].Ref.Type, "BRANCH") &&
			!strings.EqualFold(refsChanged.Changes[0].Type, "DELETE") {
			// Check if commit is associated to an open PR.
			hasOpenPR, err := router.refsChangedHasOpenPullRequest(refsChanged.Repository.Project.Key, refsChanged.Repository.Slug, refsChanged.Changes[0].ToHash)
			if err != nil {
				logger.Errorf("checking if changed branch ref has an open pull request", zap.Error(err))
				common.SendErrorResponse(writer, err.Error())
				route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
				return
			}

			// Do not publish this Branch repo:refs_changed event if a related Pull Request is already opened for the commit.
			if hasOpenPR {
				logger.Info("skipping publishing event, commit has an open pull request")
				common.SendSuccessResponse(writer, "success")
				return
			}
		}
	}

	event := &events.BitbucketServerEventData{
		Headers:  request.Header,
		Body:     (*json.RawMessage)(&body),
		Metadata: router.bitbucketserverEventSource.Metadata,
	}

	eventBody, err := json.Marshal(event)
	if err != nil {
		logger.Errorw("failed to parse event", zap.Error(err))
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

	if !bitbucketserverEventSource.DeleteHookOnFinish {
		logger.Info("not configured to delete webhooks, skipping")
		return nil
	}

	if len(router.hookIDs) == 0 {
		logger.Info("no need to delete webhooks, skipping")
		return nil
	}

	logger.Info("deleting webhooks from bitbucket")

	bitbucketRepositories := bitbucketserverEventSource.GetBitbucketServerRepositories()

	if len(bitbucketserverEventSource.Projects) > 0 {
		bitbucketProjectRepositories, err := getProjectRepositories(router.deleteClient, bitbucketserverEventSource.Projects)
		if err != nil {
			return err
		}

		bitbucketRepositories = append(bitbucketRepositories, bitbucketProjectRepositories...)
	}

	for _, repo := range bitbucketRepositories {
		id, ok := router.hookIDs[repo.ProjectKey+","+repo.RepositorySlug]
		if !ok {
			return fmt.Errorf("can not find hook ID for project-key: %s, repository-slug: %s", repo.ProjectKey, repo.RepositorySlug)
		}

		_, err := router.deleteClient.DefaultApi.DeleteWebhook(repo.ProjectKey, repo.RepositorySlug, int32(id))
		if err != nil {
			return fmt.Errorf("failed to delete bitbucketserver webhook. err: %w", err)
		}

		logger.Infow("bitbucket server webhook deleted",
			zap.String("project-key", repo.ProjectKey), zap.String("repository-slug", repo.RepositorySlug))
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

	logger.Info("retrieving the access token credentials...")
	bitbucketToken, err := common.GetSecretFromVolume(bitbucketserverEventSource.AccessToken)
	if err != nil {
		return fmt.Errorf("getting bitbucketserver token. err: %w", err)
	}

	logger.Info("setting up the client to connect to Bitbucket Server...")
	bitbucketConfig, err := newBitbucketServerClientCfg(bitbucketserverEventSource)
	if err != nil {
		return fmt.Errorf("initializing bitbucketserver client config. err: %w", err)
	}

	bitbucketURL, err := url.Parse(bitbucketserverEventSource.BitbucketServerBaseURL)
	if err != nil {
		return fmt.Errorf("parsing bitbucketserver url. err: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	bitbucketClient := newBitbucketServerClient(ctx, bitbucketConfig, bitbucketToken)
	bitbucketDeleteClient := newBitbucketServerClient(context.Background(), bitbucketConfig, bitbucketToken)

	route := webhook.NewRoute(bitbucketserverEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	router := &Router{
		route:  route,
		client: bitbucketClient,
		customClient: &customBitbucketClient{
			client: bitbucketConfig.HTTPClient,
			ctx:    ctx,
			token:  bitbucketToken,
			url:    bitbucketURL,
		},
		deleteClient:               bitbucketDeleteClient,
		bitbucketserverEventSource: bitbucketserverEventSource,
		hookIDs:                    make(map[string]int),
	}

	if !bitbucketserverEventSource.ShouldCreateWebhooks() {
		logger.Info("access token or webhook configuration were not provided, skipping webhooks creation")
		return webhook.ManageRoute(ctx, router, controller, dispatch)
	}

	if bitbucketserverEventSource.WebhookSecret != nil {
		logger.Info("retrieving the webhook secret...")
		webhookSecret, err := common.GetSecretFromVolume(bitbucketserverEventSource.WebhookSecret)
		if err != nil {
			return fmt.Errorf("getting bitbucketserver webhook secret. err: %w", err)
		}

		router.hookSecret = webhookSecret
	}

	applyWebhooks := func() {
		bitbucketRepositories := bitbucketserverEventSource.GetBitbucketServerRepositories()

		if len(bitbucketserverEventSource.Projects) > 0 {
			bitbucketProjectRepositories, err := getProjectRepositories(router.client, bitbucketserverEventSource.Projects)
			if err != nil {
				logger.Errorw("failed to apply Bitbucket webhook", zap.Error(err))
			}

			bitbucketRepositories = append(bitbucketRepositories, bitbucketProjectRepositories...)
		}

		for _, repo := range bitbucketRepositories {
			if err = router.applyBitbucketServerWebhook(repo); err != nil {
				logger.Errorw("failed to apply Bitbucket webhook",
					zap.String("project-key", repo.ProjectKey), zap.String("repository-slug", repo.RepositorySlug), zap.Error(err))
				continue
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	// When running multiple replicas of this event source, they will try to create webhooks at the same time.
	// Randomly delay running the initial apply webhooks func to mitigate the issue.
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(2000)))
	time.Sleep(time.Duration(randomNum.Int64()) * time.Millisecond)
	applyWebhooks()

	var checkInterval time.Duration
	if bitbucketserverEventSource.CheckInterval == "" {
		checkInterval = 60 * time.Second
	} else {
		checkInterval, err = time.ParseDuration(bitbucketserverEventSource.CheckInterval)
		if err != nil {
			return err
		}
	}

	go func() {
		// Another kind of race conditions might happen when pods do rolling upgrade - new pod starts
		// and old pod terminates, if DeleteHookOnFinish is true, the hook will be deleted from Bitbucket.
		// This is a workaround to mitigate the race conditions.
		logger.Info("starting bitbucket hooks manager daemon")

		ticker := time.NewTicker(checkInterval)
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
func (router *Router) applyBitbucketServerWebhook(repo v1alpha1.BitbucketServerRepository) error {
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

	formattedURL := common.FormattedURL(bitbucketserverEventSource.Webhook.URL, bitbucketserverEventSource.Webhook.Endpoint)

	hooks, err := router.listWebhooks(repo)
	if err != nil {
		return fmt.Errorf("failed to list existing hooks to check for duplicates for repository %s/%s, %w", repo.ProjectKey, repo.RepositorySlug, err)
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
		return fmt.Errorf("failed to create request body from webhook, %w", err)
	}

	// Update the webhook when it does exist and the events/configuration have changed
	if isAlreadyExists {
		logger.Info("webhook already exists")
		if router.shouldUpdateWebhook(existingHook, newHook) {
			logger.Info("webhook requires an update")
			err = router.updateWebhook(existingHook.ID, requestBody, repo)
			if err != nil {
				return fmt.Errorf("failed to update webhook. err: %w", err)
			}

			logger.With("hook-id", existingHook.ID).Info("hook successfully updated")
		}

		return nil
	}

	// Create the webhook when it doesn't exist yet
	createdHook, err := router.createWebhook(requestBody, repo)
	if err != nil {
		return fmt.Errorf("failed to create webhook. err: %w", err)
	}

	router.hookIDs[repo.ProjectKey+","+repo.RepositorySlug] = createdHook.ID

	logger.With("hook-id", createdHook.ID).Info("hook successfully registered")

	return nil
}

func (router *Router) listWebhooks(repo v1alpha1.BitbucketServerRepository) ([]bitbucketv1.Webhook, error) {
	apiResponse, err := router.client.DefaultApi.FindWebhooks(repo.ProjectKey, repo.RepositorySlug, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing hooks to check for duplicates for repository %s/%s, %w", repo.ProjectKey, repo.RepositorySlug, err)
	}

	hooks, err := bitbucketv1.GetWebhooksResponse(apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the list of webhooks for repository %s/%s, %w", repo.ProjectKey, repo.RepositorySlug, err)
	}

	return hooks, nil
}

func (router *Router) createWebhook(requestBody []byte, repo v1alpha1.BitbucketServerRepository) (*bitbucketv1.Webhook, error) {
	apiResponse, err := router.client.DefaultApi.CreateWebhook(repo.ProjectKey, repo.RepositorySlug, requestBody, []string{"application/json"})
	if err != nil {
		return nil, fmt.Errorf("failed to add webhook. err: %w", err)
	}

	var createdHook *bitbucketv1.Webhook
	err = mapstructure.Decode(apiResponse.Values, &createdHook)
	if err != nil {
		return nil, fmt.Errorf("failed to convert API response to Webhook struct. err: %w", err)
	}

	return createdHook, nil
}

func (router *Router) updateWebhook(hookID int, requestBody []byte, repo v1alpha1.BitbucketServerRepository) error {
	_, err := router.client.DefaultApi.UpdateWebhook(repo.ProjectKey, repo.RepositorySlug, int32(hookID), requestBody, []string{"application/json"})

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
			return nil, fmt.Errorf("failed to convert webhook to map, %w", err)
		}

		delete(hookMap, "configuration")

		finalHook = hookMap
	}

	requestBody, err := json.Marshal(finalHook)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new webhook to JSON, %w", err)
	}

	return requestBody, nil
}

func (router *Router) parseAndValidateBitbucketServerRequest(request *http.Request) ([]byte, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request body, %w", err)
	}

	if len(router.hookSecret) != 0 {
		signature := request.Header.Get("X-Hub-Signature")
		if len(signature) == 0 {
			return nil, fmt.Errorf("missing signature header")
		}

		mac := hmac.New(sha256.New, []byte(router.hookSecret))
		_, _ = mac.Write(body)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature[7:]), []byte(expectedMAC)) {
			return nil, fmt.Errorf("hmac verification failed")
		}
	}

	return body, nil
}

// refsChangedHasOpenPullRequest returns true if the changed commit has an open pull request
func (router *Router) refsChangedHasOpenPullRequest(project, repository, commit string) (bool, error) {
	bitbucketPullRequests, err := router.customClient.GetCommitPullRequests(project, repository, commit)
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

type bitbucketServerReposPager struct {
	Size          int                      `json:"size"`
	Limit         int                      `json:"limit"`
	Start         int                      `json:"start"`
	NextPageStart int                      `json:"nextPageStart"`
	IsLastPage    bool                     `json:"isLastPage"`
	Values        []bitbucketv1.Repository `json:"values"`
}

// getProjectRepositories returns all the Bitbucket Server repositories in the provided projects
func getProjectRepositories(client *bitbucketv1.APIClient, projects []string) ([]v1alpha1.BitbucketServerRepository, error) {
	var bitbucketRepos []bitbucketv1.Repository
	for _, project := range projects {
		paginationOptions := map[string]interface{}{"start": 0, "limit": 500}
		for {
			response, err := client.DefaultApi.GetRepositoriesWithOptions(project, paginationOptions)
			if err != nil {
				return nil, fmt.Errorf("unable to list repositories for project %s: %w", project, err)
			}

			var reposPager bitbucketServerReposPager
			err = mapstructure.Decode(response.Values, &reposPager)
			if err != nil {
				return nil, fmt.Errorf("unable to decode repositories for project %s: %w", project, err)
			}

			bitbucketRepos = append(bitbucketRepos, reposPager.Values...)

			if reposPager.IsLastPage {
				break
			}

			paginationOptions["start"] = reposPager.NextPageStart
		}
	}

	var repositories []v1alpha1.BitbucketServerRepository
	for n := range bitbucketRepos {
		repositories = append(repositories, v1alpha1.BitbucketServerRepository{
			ProjectKey:     bitbucketRepos[n].Project.Key,
			RepositorySlug: bitbucketRepos[n].Slug,
		})
	}

	return repositories, nil
}

func newBitbucketServerClientCfg(bitbucketserverEventSource *v1alpha1.BitbucketServerEventSource) (*bitbucketv1.Configuration, error) {
	bitbucketCfg := bitbucketv1.NewConfiguration(bitbucketserverEventSource.BitbucketServerBaseURL)
	bitbucketCfg.AddDefaultHeader("x-atlassian-token", "no-check")
	bitbucketCfg.AddDefaultHeader("x-requested-with", "XMLHttpRequest")
	bitbucketCfg.HTTPClient = &http.Client{}

	if bitbucketserverEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(bitbucketserverEventSource.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to get the tls configuration. err: %w", err)
		}

		bitbucketCfg.HTTPClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	return bitbucketCfg, nil
}

func newBitbucketServerClient(ctx context.Context, bitbucketConfig *bitbucketv1.Configuration, bitbucketToken string) *bitbucketv1.APIClient {
	ctx = context.WithValue(ctx, bitbucketv1.ContextAccessToken, bitbucketToken)
	return bitbucketv1.NewAPIClient(ctx, bitbucketConfig)
}

type refsChangedWebhookEvent struct {
	EventKey   string `json:"eventKey"`
	Repository struct {
		Slug    string `json:"slug"`
		Project struct {
			Key string `json:"key"`
		} `json:"project"`
	} `json:"repository"`
	Changes []struct {
		Ref struct {
			Type string `json:"type"`
		} `json:"ref"`
		ToHash string `json:"toHash"`
		Type   string `json:"type"`
	} `json:"changes"`
}

// customBitbucketClient returns a Bitbucket HTTP client that implements methods that gfleury/go-bitbucket-v1 does not.
// Specifically getting Pull Requests associated to a commit is not supported by gfleury/go-bitbucket-v1.
type customBitbucketClient struct {
	client *http.Client
	ctx    context.Context
	token  string
	url    *url.URL
}

type pagination struct {
	Start int
	Limit int
}

func (p *pagination) StartStr() string {
	return strconv.Itoa(p.Start)
}

func (p *pagination) LimitStr() string {
	return strconv.Itoa(p.Limit)
}

func (c *customBitbucketClient) authHeader(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
}

func (c *customBitbucketClient) get(u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	c.authHeader(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode > 299 {
		resBody, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			return nil, readErr
		}

		return nil, fmt.Errorf("calling endpoint '%s' failed: status %d: response body: %s", u, res.StatusCode, resBody)
	}

	return io.ReadAll(res.Body)
}

// pullRequestRes is a struct containing information about the Pull Request.
type pullRequestRes struct {
	ID    int    `json:"id"`
	State string `json:"state"`
}

// pagedPullRequestsRes is a paged response with values of pullRequestRes.
type pagedPullRequestsRes struct {
	Size          int              `json:"size"`
	Limit         int              `json:"limit"`
	IsLastPage    bool             `json:"isLastPage"`
	Values        []pullRequestRes `json:"values"`
	Start         int              `json:"start"`
	NextPageStart int              `json:"nextPageStart"`
}

// GetCommitPullRequests returns all the Pull Requests associated to the commit id.
func (c *customBitbucketClient) GetCommitPullRequests(project, repository, commit string) ([]pullRequestRes, error) {
	p := pagination{Start: 0, Limit: 500}

	commitsURL := c.url.JoinPath(fmt.Sprintf("api/1.0/projects/%s/repos/%s/commits/%s/pull-requests", project, repository, commit))
	query := commitsURL.Query()
	query.Set("limit", p.LimitStr())

	var pullRequests []pullRequestRes
	for {
		query.Set("start", p.StartStr())
		commitsURL.RawQuery = query.Encode()

		body, err := c.get(commitsURL.String())
		if err != nil {
			return nil, err
		}

		var pagedPullRequests pagedPullRequestsRes
		err = json.Unmarshal(body, &pagedPullRequests)
		if err != nil {
			return nil, err
		}

		pullRequests = append(pullRequests, pagedPullRequests.Values...)

		if pagedPullRequests.IsLastPage {
			break
		}

		p.Start = pagedPullRequests.NextPageStart
	}

	return pullRequests, nil
}
