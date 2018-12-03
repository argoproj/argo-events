package webhook

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"go.uber.org/atomic"
	"io/ioutil"
	"net/http"
	"sync"
)

var (
	// whether http server has started or not
	hasServerStarted atomic.Bool

	// srv holds reference to http server
	srv http.Server

	// mutex synchronizes activeRoutes
	mutex sync.Mutex
	// as http package does not provide method for unregistering routes,
	// this keeps track of configured http routes and their methods.
	// keeps endpoints as keys and corresponding http methods as a map
	activeRoutes = make(map[string]map[string]struct{})
)

// Runs a gateway configuration
func (ce *WebhookConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	defer func() {
		gateways.Recover()
	}()

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	w, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *w).Msg("webhook configuration")

	go ce.listenEvents(w, config)

	for {
		select {
		case _, ok :=<-config.StartChan:
			if ok {
				config.Active = true
				ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running")
			}

		case data, ok := <-config.DataChan:
			if ok {
				err := ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: data,
				})
				if err != nil {
					config.ErrChan <- err
				}
			}

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			mutex.Lock()
			activeHTTPMethods, ok := activeRoutes[w.Endpoint]
			if ok {
				delete(activeHTTPMethods, w.Method)
			}
			if w.Port != "" && hasServerStarted.Load() {
				ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping http server")
				err = srv.Shutdown(context.Background())
				if err != nil {
					ce.GatewayConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("error occurred while shutting down server")
				}
			}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			mutex.Unlock()
			config.DoneChan <- struct{}{}
			return
		}
	}
}

func (ce *WebhookConfigExecutor) listenEvents(w *webhook, config *gateways.ConfigContext) {
	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err := common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	config.StartChan <- struct{}{}

	// start a http server only if given configuration contains port information and no other
	// configuration previously started the server
	if w.Port != "" && !hasServerStarted.Load() {
		// mark http server as started
		hasServerStarted.Store(true)
		go func() {
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("http-port", w.Port).Msg("http server started listening...")
			srv := &http.Server{Addr: ":" + fmt.Sprintf("%s", w.Port)}
			err := srv.ListenAndServe()
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("http server stopped")
			if err == http.ErrServerClosed {
				err = nil
			}
			if err != nil {
				config.ErrChan <- err
				return
			}
		}()
	}

	// configure endpoint and http method
	if w.Endpoint != "" && w.Method != "" {
		if _, ok := activeRoutes[w.Endpoint]; !ok {
			mutex.Lock()
			activeRoutes[w.Endpoint] = make(map[string]struct{})
			// save event channel for this connection/configuration
			activeRoutes[w.Endpoint][w.Method] = struct{}{}
			mutex.Unlock()

			// add a handler for endpoint if not already added.
			http.HandleFunc(w.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
				// check if http methods match and route and http method is registered.
				if _, ok := activeRoutes[w.Endpoint]; ok {
					if _, isActive := activeRoutes[w.Endpoint][request.Method]; isActive {
						ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("endpoint", w.Endpoint).Str("http-method", w.Method).Msg("received a request")
						body, err := ioutil.ReadAll(request.Body)
						if err != nil {
							ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to parse request body")
							common.SendErrorResponse(writer)
						} else {
							common.SendSuccessResponse(writer)
							ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Str("payload", string(body)).Msg("payload")
							// dispatch event to gateway transformer
							config.DataChan <- body
						}
					} else {
						ce.GatewayConfig.Log.Warn().Str("config-key", config.Data.Src).Str("endpoint", w.Endpoint).Str("http-method", request.Method).Msg("endpoint and http method is not an active route")
						common.SendErrorResponse(writer)
					}
				} else {
					ce.GatewayConfig.Log.Warn().Str("config-key", config.Data.Src).Str("endpoint", w.Endpoint).Msg("endpoint is not active")
					common.SendErrorResponse(writer)
				}
			})
		} else {
			mutex.Lock()
			activeRoutes[w.Endpoint][w.Method] = struct{}{}
			mutex.Unlock()
		}

		ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running.")
	}

	<-config.DoneChan
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
	config.ShutdownChan <- struct{}{}
}
