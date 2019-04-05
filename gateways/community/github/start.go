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
	"net/http"
	"net/url"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/store"
	gh "github.com/google/go-github/github"
	corev1 "k8s.io/api/core/v1"
)

const (
	githubEventHeader    = "X-GitHub-Event"
	githubDeliveryHeader = "X-GitHub-Delivery"
)

var (
	helper = gwcommon.NewWebhookHelper()
)

func init() {
	go gwcommon.InitRouteChannels(helper)
}

// getCredentials for github
func (rc *RouteConfig) getCredentials(gs *corev1.SecretKeySelector) (*cred, error) {
	token, err := store.GetSecrets(rc.clientset, rc.namespace, gs.Name, gs.Key)
	if err != nil {
		return nil, err
	}
	return &cred{
		secret: token,
	}, nil
}

func (rc *RouteConfig) GetRoute() *gwcommon.Route {
	return rc.route
}

func (rc *RouteConfig) PostStart() error {
	gc := rc.ges

	c, err := rc.getCredentials(gc.APIToken)
	if err != nil {
		return fmt.Errorf("failed to rtrieve github credentials. err: %+v", err)
	}

	PATTransport := TokenAuthTransport{
		Token: c.secret,
	}

	formattedUrl := gwcommon.GenerateFormattedURL(gc.Hook)
	hookConfig := map[string]interface{}{
		"url": &formattedUrl,
	}

	if gc.ContentType != "" {
		hookConfig["content_type"] = gc.ContentType
	}

	if gc.Insecure {
		hookConfig["insecure_ssl"] = "1"
	} else {
		hookConfig["insecure_ssl"] = "0"
	}

	if gc.WebHookSecret != nil {
		sc, err := rc.getCredentials(gc.WebHookSecret)
		if err != nil {
			return fmt.Errorf("failed to retrieve webhook secret. err: %+v", err)
		}
		hookConfig["secret"] = sc.secret
	}

	rc.hook = &gh.Hook{
		Events: gc.Events,
		Active: gh.Bool(gc.Active),
		Config: hookConfig,
		ID:     &rc.ges.Id,
	}

	rc.client = gh.NewClient(PATTransport.Client())
	if gc.GithubBaseURL != "" {
		baseURL, err := url.Parse(gc.GithubBaseURL)
		if err != nil {
			return fmt.Errorf("failed to parse github base url. err: %s", err)
		}
		rc.client.BaseURL = baseURL
	}
	if gc.GithubUploadURL != "" {
		uploadURL, err := url.Parse(gc.GithubUploadURL)
		if err != nil {
			return fmt.Errorf("failed to parse github upload url. err: %s", err)
		}
		rc.client.UploadURL = uploadURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	hook, _, err := rc.client.Repositories.CreateHook(ctx, gc.Owner, gc.Repository, rc.hook)
	if err != nil {
		// Continue if error is because hook already exists
		er, ok := err.(*gh.ErrorResponse)
		if !ok || er.Response.StatusCode != http.StatusUnprocessableEntity {
			return fmt.Errorf("failed to create webhook. err: %+v", err)
		}
	}

	if hook == nil {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		hook, _, err = rc.client.Repositories.GetHook(ctx, gc.Owner, gc.Repository, rc.ges.Id)
		if err != nil {
			return fmt.Errorf("failed to get existing webhook with id %d. err: %+v", rc.ges.Id, err)
		}
	}

	if gc.WebHookSecret != nil {
		// As secret in hook config is masked with asterisk (*), replace it with unmasked secret.
		hook.Config["secret"] = hookConfig["secret"]
	}

	rc.hook = hook
	rc.route.Logger.WithEventSource(rc.route.EventSource.Name).Info("github hook created")
	return nil
}

// PostStop runs after event source is stopped
func (rc *RouteConfig) PostStop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := rc.client.Repositories.DeleteHook(ctx, rc.ges.Owner, rc.ges.Repository, *rc.hook.ID); err != nil {
		return fmt.Errorf("failed to delete hook. err: %+v", err)
	}
	rc.route.Logger.WithEventSource(rc.route.EventSource.Name).Info("github hook deleted")
	return nil
}

// StartEventSource starts an event source
func (ese *GithubEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	log := ese.Log.WithEventSource(eventSource.Name)

	log.Info("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}
	gc := config.(*githubEventSource)

	return gwcommon.ProcessRoute(&RouteConfig{
		route: &gwcommon.Route{
			Logger:      ese.Log,
			EventSource: eventSource,
			Webhook:     gc.Hook,
			StartCh:     make(chan struct{}),
		},
		clientset: ese.Clientset,
		namespace: ese.Namespace,
		ges:       gc,
	}, helper, eventStream)
}

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

// routeActiveHandler handles new route
func (rc *RouteConfig) RouteHandler(writer http.ResponseWriter, request *http.Request) {
	r := rc.route

	logger := r.Logger.
		WithEventSource(r.EventSource.Name).
		WithEndpoint(r.Webhook.Endpoint).
		WithPort(r.Webhook.Port)

	logger.Info("request received")

	if !helper.ActiveEndpoints[r.Webhook.Endpoint].Active {
		logger.Info("endpoint is not active")
		common.SendErrorResponse(writer, "")
		return
	}

	hook := rc.hook
	secret := ""
	if s, ok := hook.Config["secret"]; ok {
		secret = s.(string)
	}
	body, err := parseValidateRequest(request, []byte(secret))
	if err != nil {
		logger.WithError(err).Error("request is not valid event notification")
		common.SendErrorResponse(writer, "")
		return
	}

	helper.ActiveEndpoints[r.Webhook.Endpoint].DataCh <- body
	logger.Info("request successfully processed")
	common.SendSuccessResponse(writer, "")
}
