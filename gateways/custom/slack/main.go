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

package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	slackAPI "github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sync"
)

// SlackAPIType is type of api to use to listen to events.
type SlackAPIType string

// Possible values for SlackAPIType
var (
	// Refer https://api.slack.com/events-api
	HttpAPI SlackAPIType = "HTTP"

	// Refer https://api.slack.com/rtm
	RtmAPI SlackAPIType = "RTM"
)

var (
	// mutex synchronizes activeServers
	mutex sync.Mutex
	// activeServers keeps track of currently running http servers.
	activeServers = make(map[string]*http.Server)

	// mutex synchronizes activeRoutes
	routesMutex sync.Mutex
	// activeRoutes keep track of active routes for a http server
	activeRoutes = make(map[string]map[string]struct{})
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// slack implements ConfigExecutor
type slack struct{}

// slackConfig contains configuration for Slack API to consume events.
type slackConfig struct {
	// Events define types of Slack events to listen to
	// Refer https://api.slack.com/events/api
	Events []string `json:"events" protobuf:"bytes,1,opt,name=events"`

	// Endpoint on which Slack will send HTTP requests.
	// Refer https://api.slack.com/web
	Endpoint string `json:"endpoint" protobuf:"bytes,2,opt,name=endpoint"`

	// APIToken contains API access/token key to authenticate client
	APIToken SlackAPIToken `json:"apiToken" protobuf:"bytes,3,opt,name=apiToken"`

	// Type of Slack notification. Either RTM or HTTP Events
	// Refer https://api.slack.com/rtm and https://api.slack.com/events-api
	Type SlackAPIType `json:"type" protobuf:"bytes,4,opt,name=type"`

	// Port to run HTTP server on.
	Port string `json:"port,omitempty" protobuf:"bytes,5,opt,name=port"`

	// srv holds reference to http server
	srv *http.Server
	mux *http.ServeMux
}

// SlackAPIToken contains reference to k8
type SlackAPIToken struct {
	// Name of the k8 secret that holds access token
	Name string
	// Key in k8 secret for access token
	Key string
}

type Server struct {
	mux *http.ServeMux
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *slackConfig) getAccessTokens() (*corev1.Secret, error) {
	// read slack access token from k8 secret
	secret, err := gatewayConfig.Clientset.CoreV1().Secrets(gatewayConfig.Namespace).Get(s.APIToken.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// Runs a gateway configuration
func (s *slack) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	var sConfig *slackConfig
	err = yaml.Unmarshal([]byte(config.Data.Config), sConfig)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	switch sConfig.Type {
	case HttpAPI:
		go func() {
			<-config.StopCh
			config.Active = false
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")

			// remove the endpoint.
			routesMutex.Lock()
			if _, ok := activeRoutes[sConfig.Port]; ok {
				delete(activeRoutes[sConfig.Port], sConfig.Endpoint)
				// Check if the endpoint in this configuration was the last of the active endpoints for the http server.
				// If so, shutdown the server.
				if len(activeRoutes) == 0 {
					gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("all endpoint are deactivated, stopping http server")
					err = sConfig.srv.Shutdown(context.Background())
					if err != nil {
						errMessage = "failed to stop http server"
					}
				}
			}
			routesMutex.Unlock()
			wg.Done()
		}()

		// from this point, configuration is considered active.
		config.Active = true
		// generate k8 event to update configuration state in gateway.
		event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
		_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
		if err != nil {
			gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
			return err
		}

		// start a http server only if no other configuration previously started the server on given port
		mutex.Lock()
		if _, ok := activeServers[sConfig.Port]; !ok {
			gatewayConfig.Log.Info().Str("port", sConfig.Port).Msg("http server started listening...")
			s := &Server{
				mux: http.NewServeMux(),
			}
			sConfig.mux = s.mux
			sConfig.srv = &http.Server{
				Addr:    ":" + fmt.Sprintf("%s", sConfig.Port),
				Handler: s,
			}
			activeServers[sConfig.Port] = sConfig.srv

			// start http server
			go func() {
				err = sConfig.srv.ListenAndServe()
				gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("http server stopped")
				if err == http.ErrServerClosed {
					err = nil
				}
				if err != nil {
					errMessage = "http server stopped"
				}
				if config.Active == true {
					config.StopCh <- struct{}{}
				}
				return
			}()
		}
		mutex.Unlock()

		// add endpoint
		routesMutex.Lock()

		if _, ok := activeRoutes[sConfig.Port]; !ok {
			activeRoutes[sConfig.Port] = make(map[string]struct{})
		}

		if _, ok := activeRoutes[sConfig.Port][sConfig.Endpoint]; !ok {
			activeRoutes[sConfig.Port][sConfig.Endpoint] = struct{}{}
			sConfig.mux.HandleFunc(sConfig.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
				gatewayConfig.Log.Info().Str("endpoint", sConfig.Endpoint).Msg("received a slack event")
				buf := new(bytes.Buffer)
				buf.ReadFrom(request.Body)
				body := buf.String()
				eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body))
				if err != nil {
					gatewayConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to parse event")
					common.SendErrorResponse(writer)
				}
				for _, eventType := range sConfig.Events {
					if eventType == eventsAPIEvent.Type {
						gatewayConfig.Log.Info().Str("endpoint", sConfig.Endpoint).Msg("dispatching event to gateway-processor")
						common.SendSuccessResponse(writer)
						gatewayConfig.Log.Debug().Str("body", string(body)).Msg("payload")
						var buff bytes.Buffer
						enc := gob.NewEncoder(&buff)
						err := enc.Encode(eventsAPIEvent)
						if err != nil {
							gatewayConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to encode slack event")
							continue
						}
						// dispatch event to gateway transformer
						gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
							Src:     config.Data.Src,
							Payload: buff.Bytes(),
						})
					}
				}

			})
		}
		routesMutex.Unlock()

	case RtmAPI:
		// read slack access token from k8 secret
		secret, err := sConfig.getAccessTokens()
		if err != nil {
			return err
		}
		accessToken := string(secret.Data[sConfig.APIToken.Key])
		rtm := slackAPI.New(accessToken).NewRTM()
		go rtm.ManageConnection()

		for msg := range rtm.IncomingEvents {
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("received a slack event")
			var buff bytes.Buffer
			enc := gob.NewEncoder(&buff)
			err := enc.Encode(msg)
			if err != nil {
				gatewayConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to encode slack event")
			} else {
				// dispatch event to gateway transformer
				gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: buff.Bytes(),
				})
			}
		}

	default:
		errMessage = fmt.Sprintf("slack API type: %s", string(sConfig.Type))
		err = fmt.Errorf("unknow slack API type")
		return err
	}

	wg.Wait()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// Stops a configuration
func (s *slack) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

func main() {
	err := gatewayConfig.TransformerReadinessProbe()
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to connect to gateway transformer")
	}
	_, err = gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch k8 events for gateway configuration state updates")
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &slack{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}
