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
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/satori/go.uuid"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
	"github.com/joncalhoun/qson"
	"encoding/json"
	"net/url"
	"strings"
)

var (
	// mutex synchronizes activeServers
	mutex sync.Mutex
	// activeServers keeps track of currently running http servers.
	activeServers = make(map[string]*http.ServeMux)

	// mutex synchronizes activeRoutes
	routesMutex sync.Mutex
	// activeRoutes keep track of active routes for a http server
	activeRoutes = make(map[string]map[string]struct{})

	gatewayConfig = gateways.NewGatewayConfiguration()
	respBody      = `
<PublishResponse xmlns="http://argoevents-sns-server/">
    <PublishResult> 
        <MessageId>` + generateUUID().String() + `</MessageId> 
    </PublishResult> 
    <ResponseMetadata>
       <RequestId>` + generateUUID().String() + `</RequestId>
    </ResponseMetadata> 
</PublishResponse>` + "\n"
)

// storageGridConfigExecutor implements ConfigExecutor interface
type storageGridConfigExecutor struct{}

// storageGridEventConfig contains configuration for storage grid sns
type storageGridEventConfig struct {
	Port     string
	Endpoint string
	// Todo: add event and prefix filtering.
	Events []string
	Filter *Filter
	// srv holds reference to http server
	srv *http.Server
	mux *http.ServeMux
}

// Filter represents filters to apply to bucket nofifications for specifying constraints on objects
type Filter struct {
	Prefix string
	Suffix string
}

// HTTP Muxer
type server struct {
	mux *http.ServeMux
}

// ServeHTTP implementation
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// storageGridNotification is the bucket notification received from storage grid
type storageGridNotification struct {
	Action  string `json:"Action"`
	Message struct {
		Records []struct {
			EventVersion string    `json:"eventVersion"`
			EventSource  string    `json:"eventSource"`
			EventTime    time.Time `json:"eventTime"`
			EventName    string    `json:"eventName"`
			UserIdentity struct {
				PrincipalID string `json:"principalId"`
			} `json:"userIdentity"`
			RequestParameters struct {
				SourceIPAddress string `json:"sourceIPAddress"`
			} `json:"requestParameters"`
			ResponseElements struct {
				XAmzRequestID string `json:"x-amz-request-id"`
			} `json:"responseElements"`
			S3 struct {
				S3SchemaVersion string `json:"s3SchemaVersion"`
				ConfigurationID string `json:"configurationId"`
				Bucket          struct {
					Name          string `json:"name"`
					OwnerIdentity struct {
						PrincipalID string `json:"principalId"`
					} `json:"ownerIdentity"`
					Arn string `json:"arn"`
				} `json:"bucket"`
				Object struct {
					Key       string `json:"key"`
					Size      int `json:"size"`
					ETag      string `json:"eTag"`
					Sequencer string `json:"sequencer"`
				} `json:"object"`
			} `json:"s3"`
		} `json:"Records"`
	} `json:"Message"`
	TopicArn string `json:"TopicArn"`
	Version  string `json:"Version"`
}

// generateUUID returns a new uuid
func generateUUID() uuid.UUID {
	return uuid.NewV4()
}

// starts a http server
func (sgce *storageGridConfigExecutor) startHttpServer(sg *storageGridEventConfig, config *gateways.ConfigContext, err error, errMessage *string) {
	// start a http server only if no other configuration previously started the server on given port
	mutex.Lock()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("active servers", activeServers[sg.Port]).Msg("active servers")
	if _, ok := activeServers[sg.Port]; !ok {
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("port", sg.Port).Msg("http server started listening...")
		s := &server{
			mux: http.NewServeMux(),
		}
		sg.mux = s.mux
		sg.srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", sg.Port),
			Handler: s,
		}
		activeServers[sg.Port] = s.mux

		// start http server
		go func() {
			err := sg.srv.ListenAndServe()
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("http server stopped")
			if err == http.ErrServerClosed {
				err = nil
			}
			if err != nil {
				msg := fmt.Sprintf("failed to stop http server. configuration err message: %+v", err)
				errMessage = &msg
			}
			if config.Active == true {
				config.StopCh <- struct{}{}
			}
			return
		}()
	}
	mutex.Unlock()
}

// filterEvent filters notification based on event filter in a gateway configuration
func filterEvent(notification *storageGridNotification, sg *storageGridEventConfig) bool {
	if sg.Events == nil {
		return true
	}
	for _, filterEvent := range sg.Events {
		if notification.Message.Records[0].EventName == filterEvent {
			return true
		}
	}
	return false
}

// filterName filters object key based on configured prefix and/or suffix
func filterName(notification *storageGridNotification, sg *storageGridEventConfig) bool {
	if sg.Filter == nil {
		return true
	}
	if sg.Filter.Prefix != "" && sg.Filter.Suffix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Prefix) && strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Suffix)
	}
	if sg.Filter.Prefix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Prefix)
	}
	if sg.Filter.Suffix != "" {
		return strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Suffix)
	}
	return true
}

// StartConfig runs a configuration
func (sgce *storageGridConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")

	var sg *storageGridEventConfig
	err = yaml.Unmarshal([]byte(config.Data.Config), &sg)
	if err != nil {
		errMessage = "failed to parse configuration"
		return err
	}
	gatewayConfig.Log.Info().Interface("config", config.Data.Config).Interface("storage-grid", sg).Msg("configuring...")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-config.StopCh
		config.Active = false
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")

		// remove the endpoint.
		routesMutex.Lock()
		if _, ok := activeRoutes[sg.Port]; ok {
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("routes", activeRoutes[sg.Port]).Msg("active routes")
			delete(activeRoutes[sg.Port], sg.Endpoint)
			// Check if the endpoint in this configuration was the last of the active endpoints for the http server.
			// If so, shutdown the server.
			if len(activeRoutes[sg.Port]) == 0 {
				gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("all endpoint are deactivated, stopping http server")
				err = sg.srv.Shutdown(context.Background())
				if err != nil {
					// previous err message is useful when there was an error in configuration
					errMessage = fmt.Sprintf("failed to stop http server. configuration err message: %s", errMessage)
				}
			}
		}
		routesMutex.Unlock()
		wg.Done()
	}()

	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	// start a http server only if no other configuration previously started the server on given port
	sgce.startHttpServer(sg, config, err, &errMessage)

	// add endpoint
	routesMutex.Lock()
	if _, ok := activeRoutes[sg.Port]; !ok {
		activeRoutes[sg.Port] = make(map[string]struct{})
	}
	if _, ok := activeRoutes[sg.Port][sg.Endpoint]; !ok {
		activeRoutes[sg.Port][sg.Endpoint] = struct{}{}

		// server with same port is already started by another configuration
		if sg.mux == nil {
			mutex.Lock()
			sg.mux = activeServers[sg.Port]
			mutex.Unlock()
		}

		// if the configuration that started the server was removed even before we had chance to regiser this endpoint against the port,
		// and it was last endpoint for port, server is now start new http server
		if sg.mux == nil {
			sgce.startHttpServer(sg, config, err, &errMessage)
		}

		sg.mux.HandleFunc(sg.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
			gatewayConfig.Log.Info().Str("endpoint", sg.Endpoint).Str("http-method", request.Method).Msg("received a request")
			// Todo: find a better to handle route deletion
			if _, ok := activeRoutes[sg.Port][sg.Endpoint]; ok {
				body, err := ioutil.ReadAll(request.Body)
				if err != nil {
					gatewayConfig.Log.Error().Err(err).Msg("failed to parse request body")
					common.SendErrorResponse(writer)
				} else {
					gatewayConfig.Log.Info().Str("endpoint", sg.Endpoint).Str("http-method", request.Method).Msg("dispatching event to gateway-processor")
					gatewayConfig.Log.Debug().Str("payload", string(body)).Msg("payload")

					switch request.Method {
					case http.MethodPost, http.MethodPut:
						body, err := ioutil.ReadAll(request.Body)
						if err != nil {
							gatewayConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to parse request body")
						} else {
							gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("msg", string(body)).Msg("msg body")
						}
					case http.MethodHead:
						gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("method", http.MethodHead).Msg("received a request")
						respBody = ""
					}
					writer.WriteHeader(http.StatusOK)
					writer.Header().Add("Content-Type", "text/plain")
					writer.Write([]byte(respBody))

					// notification received from storage grid is url encoded.
					parsedURL, err := url.QueryUnescape(string(body))
					if err != nil {
						errMessage = "failed to perform query un-escape"
						gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg(errMessage)
						return
					}
					b, err := qson.ToJSON(parsedURL)
					if err != nil {
						errMessage = "failed to parse payload url into JSON"
						gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg(errMessage)
						return
					}

					var notification *storageGridNotification
					err = json.Unmarshal(b, &notification)
					if err != nil {
						errMessage = "failed to unmarshal notification JSON"
						gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg(errMessage)
						return
					}

					gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("notification", notification).Msg("parsed notification")
					if filterEvent(notification, sg) && filterName(notification, sg) {
						// dispatch event to gateway transformer
						gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
							Src:     config.Data.Src,
							Payload: b,
						})
					} else {
						gatewayConfig.Log.Warn().Str("config-key", config.Data.Src).Interface("notification", notification).Msg("discarding notification since it did not pass all filters")
					}
				}
			}
		})
	}
	routesMutex.Unlock()

	wg.Wait()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// StopConfig stops the configuration
func (sgce *storageGridConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

func main() {
	err := gatewayConfig.TransformerReadinessProbe()
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayTransformerConnection)
	}
	_, err = gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayEventWatch)
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &storageGridConfigExecutor{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayConfigmapWatch)
	}
	select {}
}
