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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/xanzy/go-gitlab"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"reflect"
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

// getCredentials for gitlab
func (ce *GitlabExecutor) getCredentials(gs *GitlabSecret) (*cred, error) {
	token, err := getSecrets(ce.Clientset, ce.Namespace, gs.Name, gs.Key)
	if err != nil {
		return nil, err
	}
	return &cred{
		token: token,
	}, nil
}

func (ce *GitlabExecutor) StartConfig(config *gateways.EventSourceContext) {
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("operating on configuration")
	g, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *g).Msg("gitlab configuration")

	go ce.listenEvents(g, config)

	for {
		select {
		case <-config.StartChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
			config.Active = true

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *GitlabExecutor) listenEvents(g *GitlabConfig, config *gateways.EventSourceContext) {
	c, err := ce.getCredentials(g.AccessToken)
	if err != nil {
		config.ErrChan <- err
		return
	}

	ce.GitlabClient = gitlab.NewClient(nil, c.token)
	ce.GitlabClient.SetBaseURL(g.GitlabBaseURL)

	opt := &gitlab.AddProjectHookOptions{
		URL:   &g.URL,
		Token: &c.token,
		EnableSSLVerification: &g.EnableSSLVerification,
	}

	elem := reflect.ValueOf(opt).Elem().FieldByName(string(g.Event))
	if ok := elem.IsValid(); !ok {
		config.ErrChan <- fmt.Errorf("unknown event %s", g.Event)
		return
	}

	iev := reflect.New(elem.Type().Elem())
	reflect.Indirect(iev).SetBool(true)
	elem.Set(iev)

	hook, _, err := ce.GitlabClient.Projects.AddProjectHook(g.ProjectId, opt)

	if err != nil {
		config.ErrChan <- err
		return
	}

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	ce.Log.Info().Str("config-key", config.Data.Src).Interface("hook-id", hook.ID).Msg("gitlab hook created")

	<-config.DataChan
	_, err = ce.GitlabClient.Projects.DeleteProjectHook(g.ProjectId, hook.ID)
	ce.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to delete gitlab hook")
	config.ShutdownChan <- struct{}{}
}
