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

package github

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	gh "github.com/google/go-github/v50/github"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// GitHub headers
const (
	githubEventHeader    = "X-GitHub-Event"
	githubDeliveryHeader = "X-GitHub-Delivery"
)

// controller controls the webhook operations
var (
	controller = webhook.NewController()
)

// set up the activation and inactivation channels to control the state of routes.
func init() {
	go webhook.ProcessRouteStatus(controller)
}

// getCredentials retrieves credentials for GitHub connection
func (router *Router) getCredentials(keySelector *corev1.SecretKeySelector) (*cred, error) {
	token, err := sharedutil.GetSecretFromVolume(keySelector)
	if err != nil {
		return nil, fmt.Errorf("secret not found, %w", err)
	}

	return &cred{
		secret: token,
	}, nil
}

// getAPITokenAuthStrategy return an TokenAuthStrategy initialised with
// the GitHub API token provided by the user
func (router *Router) getAPITokenAuthStrategy() (*TokenAuthStrategy, error) {
	apiTokenCreds, err := router.getCredentials(router.githubEventSource.APIToken)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve api token credentials, %w", err)
	}

	return &TokenAuthStrategy{
		Token: apiTokenCreds.secret,
	}, nil
}

// getGithubAppAuthStrategy return an AppsAuthStrategy initialised with
// the GitHub App credentials provided by the user
func (router *Router) getGithubAppAuthStrategy() (*AppsAuthStrategy, error) {
	appCreds := router.githubEventSource.GithubApp
	githubAppPrivateKey, err := router.getCredentials(appCreds.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve github app credentials, %w", err)
	}

	return &AppsAuthStrategy{
		AppID:          appCreds.AppID,
		BaseURL:        router.githubEventSource.GithubBaseURL,
		InstallationID: appCreds.InstallationID,
		PrivateKey:     githubAppPrivateKey.secret,
	}, nil
}

// chooseAuthStrategy returns an AuthStrategy based on the given credentials
func (router *Router) chooseAuthStrategy() (AuthStrategy, error) {
	es := router.githubEventSource
	switch {
	case es.HasGithubAPIToken():
		return router.getAPITokenAuthStrategy()
	case es.HasGithubAppCreds():
		return router.getGithubAppAuthStrategy()
	default:
		return nil, fmt.Errorf("none of the supported auth options were provided")
	}
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
	route := router.route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
	)

	logger.Info("received a request, processing it...")

	if !route.Active {
		logger.Info("endpoint is not active, won't process the request")
		sharedutil.SendErrorResponse(writer, "endpoint is inactive")
		return
	}

	defer func(start time.Time) {
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	request.Body = http.MaxBytesReader(writer, request.Body, route.Context.GetMaxPayloadSize())
	body, err := parseValidateRequest(request, []byte(router.hookSecret))
	if err != nil {
		logger.Errorw("request is not valid event notification, discarding it", zap.Error(err))
		sharedutil.SendErrorResponse(writer, err.Error())
		return
	}

	event := &events.GithubEventData{
		Headers:  request.Header,
		Body:     (*json.RawMessage)(&body),
		Metadata: router.githubEventSource.Metadata,
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
	githubEventSource := router.githubEventSource

	if githubEventSource.NeedToCreateHooks() && githubEventSource.DeleteHookOnFinish {
		logger := router.route.Logger
		logger.Info("deleting GitHub org hooks...")

		for _, org := range githubEventSource.Organizations {
			id, ok := router.orgHookIDs[org]
			if !ok {
				return fmt.Errorf("can not find hook ID for organization %s", org)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := router.githubClient.Organizations.DeleteHook(ctx, org, id); err != nil {
				return fmt.Errorf("failed to delete hook for organization %s. err: %w", org, err)
			}
			logger.Infof("GitHub hook deleted for organization %s", org)
		}

		logger.Info("deleting GitHub repo hooks...")

		for _, r := range githubEventSource.GetOwnedRepositories() {
			for _, n := range r.Names {
				id, ok := router.repoHookIDs[r.Owner+","+n]
				if !ok {
					return fmt.Errorf("can not find hook ID for repo %s/%s", r.Owner, n)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if _, err := router.githubClient.Repositories.DeleteHook(ctx, r.Owner, n, id); err != nil {
					return fmt.Errorf("failed to delete hook for repo %s/%s. err: %w", r.Owner, n, err)
				}
				logger.Infof("GitHub hook deleted for repo %s/%s", r.Owner, n)
			}
		}
	}
	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	logger := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	logger.Info("started processing the Github event source...")

	githubEventSource := &el.GithubEventSource
	route := webhook.NewRoute(githubEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	router := &Router{
		route:             route,
		githubEventSource: githubEventSource,
	}
	logger.Info("retrieving webhook secret credentials if any ...")
	if githubEventSource.WebhookSecret != nil {
		webhookSecretCreds, err := router.getCredentials(githubEventSource.WebhookSecret)
		if err != nil {
			return fmt.Errorf("failed to retrieve webhook secret. err: %w", err)
		}
		router.hookSecret = webhookSecretCreds.secret
	}

	if githubEventSource.NeedToCreateHooks() {
		// create webhooks

		// In order to successfully setup a GitHub hook for the given repository,
		// 1. Parse and validate base and upload url if provided
		// 2. Get the GitHub auth credentials and Webhook secret from K8s secrets
		// 3. Configure the hook with url, content type, ssl etc.
		// 4. Set up a GitHub client
		// 5. Set the base and upload url for the client
		// 6. Create the hook if one doesn't exist already. If exists already, then use that one.

		baseURL, err := parseUrlWithSlash(&githubEventSource.GithubBaseURL)
		if err != nil {
			return fmt.Errorf("failed to parse github base url. err: %v", err)
		}
		uploadURL, err := parseUrlWithSlash(&githubEventSource.GithubUploadURL)
		if err != nil {
			return fmt.Errorf("failed to parse github upload url. err: %v", err)
		}

		logger.Info("choosing github auth strategy...")
		authStrategy, err := router.chooseAuthStrategy()
		if err != nil {
			return fmt.Errorf("failed to get github auth strategy, %w", err)
		}

		logger.Info("setting up auth transport for http client with the chosen strategy...")
		authTransport, err := authStrategy.AuthTransport()
		if err != nil {
			return fmt.Errorf("failed to set up auth transport for http client, %w", err)
		}

		logger.Info("configuring GitHub hook...")
		formattedURL := sharedutil.FormattedURL(githubEventSource.Webhook.URL, githubEventSource.Webhook.Endpoint)
		hookConfig := map[string]interface{}{
			"url": &formattedURL,
		}
		if githubEventSource.ContentType != "" {
			hookConfig["content_type"] = githubEventSource.ContentType
		}
		if githubEventSource.Insecure {
			hookConfig["insecure_ssl"] = "1"
		} else {
			hookConfig["insecure_ssl"] = "0"
		}
		if router.hookSecret != "" {
			hookConfig["secret"] = router.hookSecret
		}

		logger.Info("setting up client for GitHub...")
		client := gh.NewClient(&http.Client{Transport: authTransport})
		if baseURL != nil && uploadURL != nil {
			logger.Info("setting up client for GitHub Enterprise...")
			client.BaseURL = baseURL
			client.UploadURL = uploadURL
		}
		logger.Infof("client set for baseURL=[%s] uploadURL=[%s]", client.BaseURL, client.UploadURL)

		router.githubClient = client
		router.repoHookIDs = make(map[string]int64)
		router.orgHookIDs = make(map[string]int64)

		hook := &gh.Hook{
			Events: githubEventSource.Events,
			Active: gh.Bool(githubEventSource.Active),
			Config: hookConfig,
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		f := func() {
			for _, org := range githubEventSource.Organizations {
				hooks, _, err := router.githubClient.Organizations.ListHooks(ctx, org, nil)
				if err != nil {
					logger.Errorf("failed to list existing webhooks of organization %s. err: %+v", org, err)
					continue
				}
				h := getHook(hooks, formattedURL, githubEventSource.Events)
				if h != nil {
					router.orgHookIDs[org] = *h.ID
					continue
				}
				logger.Infof("hook not found for organization %s, creating ...", org)
				h, _, err = router.githubClient.Organizations.CreateHook(ctx, org, hook)
				if err != nil {
					logger.Errorf("failed to create github webhook for organization %s. err: %+v", org, err)
					continue
				}
				router.orgHookIDs[org] = *h.ID
				time.Sleep(500 * time.Millisecond)
			}

			for _, r := range githubEventSource.GetOwnedRepositories() {
				for _, name := range r.Names {
					hooks, _, err := router.githubClient.Repositories.ListHooks(ctx, r.Owner, name, nil)
					if err != nil {
						logger.Errorf("failed to list existing webhooks of %s/%s. err: %+v", r.Owner, name, err)
						continue
					}
					h := getHook(hooks, formattedURL, githubEventSource.Events)
					if h != nil {
						router.repoHookIDs[r.Owner+","+name] = *h.ID
						continue
					}
					logger.Infof("hook not found for %s/%s, creating ...", r.Owner, name)
					h, _, err = router.githubClient.Repositories.CreateHook(ctx, r.Owner, name, hook)
					if err != nil {
						logger.Errorf("failed to create github webhook for %s/%s. err: %+v", r.Owner, name, err)
						continue
					}
					router.repoHookIDs[r.Owner+","+name] = *h.ID
					time.Sleep(500 * time.Millisecond)
				}
			}
		}

		// Github can not handle race conditions well - it might create multiple hooks with same config
		// when replicas > 1
		// Randomly sleep some time to mitigate the issue.
		randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(2000)))
		time.Sleep(time.Duration(randomNum.Int64()) * time.Millisecond)
		f()

		go func() {
			// Another kind of race conditions might happen when pods do rolling upgrade - new pod starts
			// and old pod terminates, if DeleteHookOnFinish is true, the hook will be deleted from github.
			// This is a workaround to mitigate the race conditions.
			logger.Info("starting github hooks manager daemon")
			for i := 0; i < 10; i++ {
				time.Sleep(60 * time.Second)
				f()
			}
			logger.Info("exiting github hooks manager daemon")
		}()
	} else {
		logger.Info("no need to create webhooks")
	}

	return webhook.ManageRoute(ctx, router, controller, dispatch)
}

// parseValidateRequest parses a http request and checks if it is valid GitHub notification
func parseValidateRequest(r *http.Request, secret []byte) ([]byte, error) {
	body, err := gh.ValidatePayload(r, secret)
	if err != nil {
		return nil, err
	}

	payload := make(map[string]interface{})
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	for _, h := range []string{
		githubEventHeader,
		githubDeliveryHeader,
	} {
		payload[h] = r.Header.Get(h)
	}
	return json.Marshal(payload)
}

// parseUrlWithSlash parses URL and enforces trailing slash expected by GitHub client
func parseUrlWithSlash(urlStr *string) (*url.URL, error) {
	if *urlStr == "" {
		return nil, nil
	}
	if !strings.HasSuffix(*urlStr, "/") {
		*urlStr += "/"
	}
	return url.Parse(*urlStr)
}
