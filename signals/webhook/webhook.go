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
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
)

const (
	EventType            string = "Webhook"
	HeaderKeyContentType string = "Content-Type"
)

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
// webhooks are special in that they should only have one http Server since the port is fixed at runtime
// this means that webhook signals are stateful, however losing this state is not a concern since
// the connections will be re-initialized by the SignalClient
type webhook struct {
	srv *http.Server
	sync.RWMutex
	inactiveEndpoints map[string]struct{}
}

// New creates a new webhook listener for the specified port
func New(port int) sdk.Listener {
	srv := &http.Server{
		Addr: fmt.Sprintf(":%v", port),
		// Good practice to enforce timeouts to avoid Slowloris attacks
		WriteTimeout: time.Second * 5,
		ReadTimeout:  time.Second * 5,
		IdleTimeout:  time.Second * 30,
	}
	// Start http server
	go func() {
		log.Printf("starting http server listening on: %s", srv.Addr)
		err := srv.ListenAndServe()
		if err == http.ErrServerClosed {
			log.Printf("successfully shutdown http server")
		} else {
			log.Panicf("http server encountered error listening: %v", err)
		}
	}()
	return &webhook{
		srv:               srv,
		inactiveEndpoints: make(map[string]struct{}),
	}
}

// this is a helper middleware to allow "deregistering" http routes
// todo: make sure that this map lookup doesn't hurt performance if there are many inactive routes
func (web *webhook) checkActiveEndpoint(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		web.RLock()
		if _, inactive := web.inactiveEndpoints[r.URL.Path]; inactive {
			web.RUnlock()
			w.WriteHeader(http.StatusNotFound)
			return
		}
		web.RUnlock()
		h.ServeHTTP(w, r)
	})
}

func (web *webhook) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	method := signal.Webhook.Method
	endpoint := signal.Webhook.Endpoint
	events := make(chan *v1alpha1.Event)

	handler := func(w http.ResponseWriter, req *http.Request) {
		log.Printf("signal '%s' received a %s request from '%s'", signal.Name, req.Method, req.Host)
		if req.Method == method {
			payload, err := ioutil.ReadAll(req.Body)
			if err != nil {
				log.Printf("unable to process request payload: %s", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			event := &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					EventType:          EventType,
					EventTypeVersion:   req.Proto,
					CloudEventsVersion: sdk.CloudEventsVersion,
					Source: &v1alpha1.URI{
						Scheme: req.RequestURI,
						Host:   req.Host,
					},
					ContentType: req.Header.Get(HeaderKeyContentType),
					EventTime:   metav1.Time{Time: time.Now().UTC()},
				},
				Data: payload,
			}
			events <- event
			w.WriteHeader(http.StatusOK)
		} else {
			log.Printf("http method '%s' does not match expected method '%s' for signal '%s'", req.Method, method, signal.Name)
			w.WriteHeader(http.StatusBadRequest)
		}
	}

	// Attach new mux handler
	// TODO: explore in tests how listening on multiple webhook signals fares...
	// TODO: use github.com/gorilla/mux
	// - pattern matching specifics
	// - same endpoint resolution - are both signals resolved?
	h := http.HandlerFunc(handler)
	http.Handle(endpoint, web.checkActiveEndpoint(h))

	// wait for stop signal
	go func() {
		defer close(events)
		<-done
		// "deregister" the handler by modifying the inactive map
		web.Lock()
		web.inactiveEndpoints[endpoint] = struct{}{}
		web.Unlock()
		log.Printf("signal '%s' stopped listening at [%s]", signal.Name, signal.Webhook.Endpoint)
	}()
	log.Printf("signal '%s' listening for webhooks at [%s]...", signal.Name, signal.Webhook.Endpoint)
	return events, nil
}
