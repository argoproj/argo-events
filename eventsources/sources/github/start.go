/*
Copyright 2018 KompiTech GmbH

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
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	gh "github.com/google/go-github/v31/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/events"
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

// getCredentials for retrieves credentials for GitHub connection
func (router *Router) getCredentials(keySelector *corev1.SecretKeySelector) (*cred, error) {
	token, err := common.GetSecretFromVolume(keySelector)
	if err != nil {
		return nil, errors.Wrap(err, "token not founnd")
	}
	return &cred{
		secret: token,
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
	route := router.route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
	)

	logger.Info("received a request, processing it...")

	if !route.Active {
		logger.Info("endpoint is not active, won't process the request")
		common.SendErrorResponse(writer, "endpoint is inactive")
		return
	}

	defer func(start time.Time) {
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	body, err := parseValidateRequest(request, []byte(router.hookSecret))
	if err != nil {
		logger.Errorw("request is not valid event notification, discarding it", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
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
	githubEventSource := router.githubEventSource

	if githubEventSource.NeedToCreateHooks() && githubEventSource.DeleteHookOnFinish {
		logger := router.route.Logger
		logger.Info("deleting GitHub hook...")

		for _, r := range githubEventSource.GetOwnedRepositories() {
			for _, n := range r.Names {
				id, ok := router.hookIDs[r.Owner+","+n]
				if !ok {
					return errors.Errorf("can not find hook ID for repo %s/%s", r.Owner, n)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if _, err := router.githubClient.Repositories.DeleteHook(ctx, r.Owner, n, id); err != nil {
					return errors.Errorf("failed to delete hook for repo %s/%s. err: %+v", r.Owner, n, err)
				}
				logger.Infof("GitHub hook deleted for repo %s/%s", r.Owner, n)
			}
		}
	}
	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
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
			return errors.Errorf("failed to retrieve webhook secret. err: %+v", err)
		}
		router.hookSecret = webhookSecretCreds.secret
	}

	if githubEventSource.NeedToCreateHooks() {
		// create webhooks

		// In order to successfully setup a GitHub hook for the given repository,
		// 1. Get the API Token and Webhook secret from K8s secrets
		// 2. Configure the hook with url, content type, ssl etc.
		// 3. Set up a GitHub client
		// 4. Set the base and upload url for the client
		// 5. Create the hook if one doesn't exist already. If exists already, then use that one.

		logger.Info("retrieving api token credentials...")
		apiTokenCreds, err := router.getCredentials(githubEventSource.APIToken)
		if err != nil {
			return errors.Errorf("failed to retrieve api token credentials. err: %+v", err)
		}

		logger.Info("setting up auth with api token...")
		PATTransport := TokenAuthTransport{
			Token: apiTokenCreds.secret,
		}

		logger.Info("configuring GitHub hook...")
		formattedURL := common.FormattedURL(githubEventSource.Webhook.URL, githubEventSource.Webhook.Endpoint)
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
		client := gh.NewClient(PATTransport.Client())

		logger.Info("setting up base url for GitHub client...")
		if githubEventSource.GithubBaseURL != "" {
			baseURL, err := url.Parse(githubEventSource.GithubBaseURL)
			if err != nil {
				return fmt.Errorf("failed to parse github base url. err: %v", err)
			}
			client.BaseURL = baseURL
		}

		logger.Info("setting up the upload url for GitHub client...")
		if githubEventSource.GithubUploadURL != "" {
			uploadURL, err := url.Parse(githubEventSource.GithubUploadURL)
			if err != nil {
				return fmt.Errorf("failed to parse github upload url. err: %v", err)
			}
			client.UploadURL = uploadURL
		}
		router.githubClient = client
		router.hookIDs = make(map[string]int64)

		hook := &gh.Hook{
			Events: githubEventSource.Events,
			Active: gh.Bool(githubEventSource.Active),
			Config: hookConfig,
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		f := func() {
			for _, r := range githubEventSource.GetOwnedRepositories() {
				for _, name := range r.Names {
					hooks, _, err := router.githubClient.Repositories.ListHooks(ctx, r.Owner, name, nil)
					if err != nil {
						logger.Errorf("failed to list existing webhooks of %s/%s. err: %+v", r.Owner, name, err)
						continue
					}
					h := getHook(hooks, formattedURL, githubEventSource.Events)
					if h != nil {
						router.hookIDs[r.Owner+","+name] = *h.ID
						continue
					}
					logger.Infof("hook not found for %s/%s, creating ...", r.Owner, name)
					h, _, err = router.githubClient.Repositories.CreateHook(ctx, r.Owner, name, hook)
					if err != nil {
						logger.Errorf("failed to create github webhook for %s/%s. err: %+v", r.Owner, name, err)
						continue
					}
					router.hookIDs[r.Owner+","+name] = *h.ID
					time.Sleep(500 * time.Millisecond)
				}
			}
		}

		// Github can not handle race conditions well - it might create multiple hooks with same config
		// when replicas > 1
		// Randomly sleep some time to mitigate the issue.
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		time.Sleep(time.Duration(r1.Intn(2000)) * time.Millisecond)
		f()

		go func() {
			// Another kind of race conditions might happen when pods do rolling upgrade - new pod starts
			// and old pod terminates, if DeleteHookOnFinish is true, the hook will be deleted from github.
			// This is a workround to mitigate the race conditions.
			logger.Info("starting github hooks manager daemon")
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					logger.Info("exiting github hooks manager daemon")
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
