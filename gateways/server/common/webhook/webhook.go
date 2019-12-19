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

package webhook

import (
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
)

// NewController returns a webhook controller
func NewController() *Controller {
	return &Controller{
		ActiveRoutes:         make(map[string]*Route),
		ActiveServerHandlers: make(map[string]*mux.Router),
		RouteActivateChan:    make(chan Router),
		RouteDeactivateChan:  make(chan Router),
	}
}

// NewRoute returns a vanilla route
func NewRoute(hookContext *Context, logger *logrus.Logger, eventSource *gateways.EventSource) *Route {
	return &Route{
		Context:     hookContext,
		Logger:      logger,
		EventSource: eventSource,
		Active:      false,
		DataCh:      make(chan []byte),
		StartCh:     make(chan struct{}),
	}
}

// ProcessRouteStatus processes route status as active and inactive.
func ProcessRouteStatus(ctrl *Controller) {
	for {
		select {
		case router := <-ctrl.RouteActivateChan:
			// start server if it has not been started on this port
			startServer(router, ctrl)
			// to allow route process incoming requests
			router.GetRoute().StartCh <- struct{}{}

		case router := <-ctrl.RouteDeactivateChan:
			router.GetRoute().Active = false
		}
	}
}

// starts a http server
func startServer(router Router, controller *Controller) {
	// start a http server only if no other configuration previously started the server on given port
	Lock.Lock()
	route := router.GetRoute()
	if _, ok := controller.ActiveServerHandlers[route.Context.Port]; !ok {
		handler := mux.NewRouter()
		server := &http.Server{
			Addr:    fmt.Sprintf(":%s", route.Context.Port),
			Handler: handler,
		}

		controller.ActiveServerHandlers[route.Context.Port] = handler

		// start http server
		go func() {
			var err error
			if route.Context.ServerCertPath == "" || route.Context.ServerKeyPath == "" {
				err = server.ListenAndServe()
			} else {
				err = server.ListenAndServeTLS(route.Context.ServerCertPath, route.Context.ServerKeyPath)
			}
			route.Logger.WithField(common.LabelEventSource, route.EventSource.Name).WithError(err).Error("http server stopped")
			if err != nil {
				route.Logger.WithError(err).WithField("port", route.Context.Port).Errorln("failed to listen and serve")
			}
		}()
	}

	// if route is not previously initialized, then assign a router against it
	if !route.initialized {
		handler := controller.ActiveServerHandlers[route.Context.Port]
		handler.HandleFunc(route.Context.Endpoint, router.HandleRoute).Methods(route.Context.Method)
	}

	Lock.Unlock()
}

// activateRoute activates a route to process incoming requests
func activateRoute(router Router, controller *Controller) {
	route := router.GetRoute()
	endpoint := route.Context.Endpoint
	// change status of route as a active route
	controller.RouteActivateChan <- router

	// wait for any route to become ready
	// if this is the first route that is added for a server, then controller will
	// start a http server before marking the route as ready
	<-route.StartCh

	log := route.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: route.EventSource.Name,
			common.LabelPort:        route.Context.Port,
			common.LabelEndpoint:    endpoint,
		})

	log.Info("activating the route...")
	route.Active = true
	log.Info("route is activated")
}

// manageRouteStream consumes data from route's data channel and stops the processing when the event source is stopped/removed
func manageRouteStream(router Router, controller *Controller, eventStream gateways.Eventing_StartEventSourceServer) error {
	route := router.GetRoute()

	for {
		select {
		case data := <-route.DataCh:
			route.Logger.WithField(common.LabelEventSource, route.EventSource.Name).Info("new event received, dispatching to gateway client")
			err := eventStream.Send(&gateways.Event{
				Name:    route.EventSource.Name,
				Payload: data,
			})
			if err != nil {
				route.Logger.WithField(common.LabelEventSource, route.EventSource.Name).WithError(err).Error("failed to send event")
				return err
			}

		case <-eventStream.Context().Done():
			route.Logger.WithField(common.LabelEventSource, route.EventSource.Name).Info("connection is closed by client")
			controller.RouteDeactivateChan <- router
			return nil
		}
	}
}

// ManagerRoute manages the lifecycle of a route
func ManageRoute(router Router, controller *Controller, eventStream gateways.Eventing_StartEventSourceServer) error {
	route := router.GetRoute()

	logger := route.Logger.WithField(common.LabelEventSource, route.EventSource.Name)

	// in order to process a route, it needs to go through
	// 1. validation - basic configuration checks
	// 2. activation - associate http handler if not done previously
	// 3. post start operations - operations that must be performed after route has been activated and ready to process requests
	// 4. consume data from route's data channel
	// 5. post stop operations - operations that must be performed after route is inactivated

	logger.Info("validating the route...")
	if err := validateRoute(router.GetRoute()); err != nil {
		logger.WithError(err).Error("route is invalid, won't initialize it")
		return err
	}

	logger.Info("activating the route...")
	activateRoute(router, controller)

	logger.Info("running operations post route activation...")
	if err := router.PostActivate(); err != nil {
		logger.WithError(err).Error("error occurred while performing post route activation operations")
		return err
	}

	logger.Info("listening to payloads for the route...")
	if err := manageRouteStream(router, controller, eventStream); err != nil {
		logger.WithError(err).Error("error occurred in consuming payload from the route")
		return err
	}

	logger.Info("running operations post route inactivation...")
	if err := router.PostInactivate(); err != nil {
		logger.WithError(err).Error("error occurred while running operations post route inactivation")
	}

	return nil
}
