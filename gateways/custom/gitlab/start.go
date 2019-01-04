/*
Copyright 2018 BlackRock, Inc.

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

package gitlab

import (
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/xanzy/go-gitlab"
	"reflect"
)

// getCredentials for gitlab
func (ce *GitlabExecutor) getCredentials(gs *GitlabSecret) (*cred, error) {
	token, err := common.GetSecret(ce.Clientset, ce.Namespace, gs.Name, gs.Key)
	if err != nil {
		return nil, err
	}
	return &cred{
		token: token,
	}, nil
}

// StartEventSource starts an event source
func (ce *GitlabExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ce.GatewayConfig.Log.Info().Str("event-source-name", *eventSource.Name).Msg("operating on event source")
	g, err := parseEventSource(eventSource.Data)
	if err != nil {
		return fmt.Errorf("%s, err: %+v", gateways.ErrEventSourceParseFailed, err)
	}
	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ce.listenEvents(g, eventSource, dataCh, errorCh, doneCh)

	return gateways.ConsumeEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ce.Log)
}

func (ce *GitlabExecutor) listenEvents(g *GitlabConfig, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	c, err := ce.getCredentials(g.AccessToken)
	if err != nil {
		errorCh <- err
		return
	}

	ce.GitlabClient = gitlab.NewClient(nil, c.token)
	if err = ce.GitlabClient.SetBaseURL(g.GitlabBaseURL); err != nil {
		errorCh <- err
		return
	}

	opt := &gitlab.AddProjectHookOptions{
		URL:   &g.URL,
		Token: &c.token,
		EnableSSLVerification: &g.EnableSSLVerification,
	}

	elem := reflect.ValueOf(opt).Elem().FieldByName(string(g.Event))
	if ok := elem.IsValid(); !ok {
		errorCh <- fmt.Errorf("unknown event %s", g.Event)
		return
	}

	iev := reflect.New(elem.Type().Elem())
	reflect.Indirect(iev).SetBool(true)
	elem.Set(iev)

	hook, _, err := ce.GitlabClient.Projects.AddProjectHook(g.ProjectId, opt)

	if err != nil {
		errorCh <- err
		return
	}

	ce.Log.Info().Str("event-source-name", *eventSource.Name).Interface("hook-id", hook.ID).Msg("gitlab hook created")

	<-doneCh
	_, err = ce.GitlabClient.Projects.DeleteProjectHook(g.ProjectId, hook.ID)
}
