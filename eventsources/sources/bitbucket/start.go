package bitbucket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/events"
	gb "github.com/ktrysmt/go-bitbucket"
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

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Desugar().Error("failed to parse request body", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
		return
	}

	event := &events.BitBucketEventData{
		Headers:  request.Header,
		Body:     (*json.RawMessage)(&body),
		Metadata: router.bitbucketEventSource.Metadata,
	}

	eventBody, err := json.Marshal(event)
	if err != nil {
		logger.Info("failed to marshal event")
		common.SendErrorResponse(writer, "invalid event")
		return
	}

	logger.Info("dispatching event on route's data channel")
	route.DataCh <- eventBody

	logger.Info("request successfully processed")
	common.SendSuccessResponse(writer, "success")
}

// PostActivate performs operations once the route is activated and ready to consume requests
func (router *Router) PostActivate() error {
	route := router.GetRoute()
	bitbucketEventSource := router.bitbucketEventSource

	// In order to set up a hook for the Bitbucket project,
	// 1. Get the creds for client
	// 2. Set up Bitbucket client
	// 3. Configure Hook with given event type
	// 4. Create project hook

	log := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
		"repo-slug", bitbucketEventSource.RepoSlug,
	)

	var client *gb.Client

	if bitbucketEventSource.Auth.Token != nil {
		log.Info("retrieving the access token credentials...")
		token, err := common.GetSecretFromVolume(bitbucketEventSource.Auth.Token)
		if err != nil {
			log.Errorw("failed to retrieve the api token", zap.Error(err))
			return err
		}
		client = gb.NewOAuthbearerToken(token)
	}
	if bitbucketEventSource.Auth.Basic != nil {
		log.Info("retrieving the basic auth credentials...")
		username, err := common.GetSecretFromVolume(bitbucketEventSource.Auth.Basic.Username)
		if err != nil {
			log.Errorw("failed to retrieve the username", zap.Error(err))
			return err
		}
		password, err := common.GetSecretFromVolume(bitbucketEventSource.Auth.Basic.Password)
		if err != nil {
			log.Errorw("failed to retrieve the password", zap.Error(err))
			return err
		}
		client = gb.NewBasicAuth(username, password)
	}

	client.SetApiBaseURL(bitbucketEventSource.ApiBaseURL)
	router.client = client

	log.Info("listing existing webhooks...")
	hooksResponse, err := client.Repositories.Webhooks.Gets(&gb.WebhooksOptions{
		Owner:    bitbucketEventSource.Owner,
		RepoSlug: bitbucketEventSource.RepoSlug,
		Active:   true,
	})
	if err != nil {
		log.Errorw("failed to list webhooks", zap.Error(err))
		return err
	}

	hooks, ok := hooksResponse.(*PaginatedWebhookSubscriptions)
	if !ok {
		log.Errorw("failed to parse the list webhooks response", zap.Any("response", hooksResponse))
		return fmt.Errorf("failed to parse the list webhooks response")
	}

	var resultHook *WebhookSubscription
	var isUpdate bool

	if hooks.Size == 0 {
		for _, hook := range hooks.Values {
			if hook.Url == bitbucketEventSource.Webhook.URL {
				resultHook = &hook
				isUpdate = true
				break
			}
		}
	}

	if resultHook == nil {
		resultHook = &WebhookSubscription{
			Url:         bitbucketEventSource.Webhook.URL,
			Description: "webhook managed by Argo-Events",
			Active:      true,
		}
	}
	resultHook.Events = bitbucketEventSource.Events

	opt := &gb.WebhooksOptions{
		Owner:       bitbucketEventSource.Owner,
		RepoSlug:    bitbucketEventSource.RepoSlug,
		Uuid:        resultHook.Uuid,
		Description: resultHook.Description,
		Url:         resultHook.Url,
		Active:      resultHook.Active,
		Events:      resultHook.Events,
	}

	if isUpdate {
		if _, err = client.Repositories.Webhooks.Update(opt); err != nil {
			log.Errorw("failed to update existing webhook", zap.Error(err))
			return err
		}
		log.Info("successfully updated the existing webhook")
		return nil
	}

	if _, err = client.Repositories.Webhooks.Create(opt); err != nil {
		log.Errorw("failed to created new webhook", zap.Error(err))
		return err
	}

	log.Info("successfully created a new webhook")
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (router *Router) PostInactivate() error {
	route := router.GetRoute()
	bitbucketEventSource := router.bitbucketEventSource
	client := router.client

	log := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
		"repo-slug", bitbucketEventSource.RepoSlug,
		"hook-uuid", router.repoUuid,
	)

	if client != nil {
		log.Info("deleting webhook...")
		if _, err := client.Repositories.Webhooks.Delete(&gb.WebhooksOptions{
			Owner:    bitbucketEventSource.Owner,
			RepoSlug: bitbucketEventSource.RepoSlug,
			Uuid:     router.repoUuid,
		}); err != nil {
			log.Errorw("failed to delete the webhook")
			return err
		}
		log.Info("successfully deleted the webhook")
	}
	return nil
}
