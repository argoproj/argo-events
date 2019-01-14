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
	"fmt"
	"time"

	"github.com/argoproj/argo-events/gateways"
	gh "github.com/google/go-github/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getSecrets retrieves the secret value from the secret in namespace with name and key
func getSecrets(client kubernetes.Interface, namespace string, name, key string) (string, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("secret '%s' does not have the key '%s'", name, key)
	}
	return string(val), nil
}

// getCredentials for github
func (ese *GithubEventSourceExecutor) getCredentials(gs *GithubSecret) (*cred, error) {
	secret, err := getSecrets(ese.Clientset, ese.Namespace, gs.Name, gs.Key)
	if err != nil {
		return nil, err
	}
	return &cred{
		secret: secret,
	}, nil
}

// StartEventSource starts an event source
func (ese *GithubEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", *eventSource.Name).Msg("operating on event source")
	g, err := parseEventSource(eventSource.Data)
	if err != nil {
		return fmt.Errorf("%s, err: %+v", gateways.ErrEventSourceParseFailed, err)
	}
	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(g, eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
}

func (ese *GithubEventSourceExecutor) listenEvents(g *GithubConfig, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	c, err := ese.getCredentials(g.APIToken)
	if err != nil {
		errorCh <- err
		return
	}

	PATTransport := TokenAuthTransport{
		Token: c.secret,
	}

	hookConfig := map[string]interface{}{
		"url": &g.URL,
	}

	if g.ContentType != "" {
		hookConfig["content_type"] = g.ContentType
	}

	if g.Insecure {
		hookConfig["insecure_ssl"] = "1"
	} else {
		hookConfig["insecure_ssl"] = "0"
	}

	if g.WebHookSecret != nil {
		sc, err := ese.getCredentials(g.WebHookSecret)
		if err != nil {
			errorCh <- err
			return
		}
		hookConfig["secret"] = sc.secret
	}

	hookSetup := &gh.Hook{
		Events: g.Events,
		Active: gh.Bool(g.Active),
		Config: hookConfig,
	}

	client := gh.NewClient(PATTransport.Client())
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	hook, _, err := client.Repositories.CreateHook(ctx, g.Owner, g.Repository, hookSetup)
	if err != nil {
		errorCh <- err
		return
	}

	ese.Log.Info().Str("event-source-name", *eventSource.Name).Interface("hook-id", *hook.ID).Msg("github hook created")

	<-doneCh
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	if _, err = client.Repositories.DeleteHook(ctx, g.Owner, g.Repository, *hook.ID); err != nil {
		ese.Log.Error().Err(err).Str("event-source-name", *eventSource.Name).Msg("failed to delete github hook")
	} else {
		ese.Log.Info().Str("event-source-name", *eventSource.Name).Interface("hook-id", *hook.ID).Msg("github hook deleted")
	}
}
