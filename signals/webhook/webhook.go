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
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
)

const (
	EventType = "Webhook"
)

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
type webhook struct{}

// New creates a new webhook listener
func New() sdk.Listener {
	return new(webhook)
}

func (*webhook) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	method := signal.Webhook.Method
	port := strconv.Itoa(int(signal.Webhook.Port))
	endpoint := signal.Webhook.Endpoint

	events := make(chan *v1alpha1.Event)

	handler := func(w http.ResponseWriter, req *http.Request) {
		log.Printf("signal '%s' received a request from '%s'", signal.Name, req.Host)
		if req.Method == method {
			payload, err := ioutil.ReadAll(req.Body)
			if err != nil {
				log.Printf("unable to process request payload: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
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
					EventTime: metav1.Time{Time: time.Now().UTC()},
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
	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, handler)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	// Start http server
	go func() {
		err := srv.ListenAndServe()
		if err == http.ErrServerClosed {
			log.Printf("successfully shutdown http server for signal '%s'", signal.Name)
		} else {
			log.Panicf("http server encountered error listening for signal '%s': %v", signal.Name, err)
		}
	}()

	// wait for stop signal
	go func() {
		defer close(events)
		<-done
		err := srv.Shutdown(context.TODO())
		if err != nil {
			log.Panicf("failed to gracefully shutdown http server for signal '%s': %s", signal.Name, err)
		}
	}()
	log.Printf("signal '%s' listening for webhooks at [%s]...", signal.Name, signal.Webhook.Endpoint)
	return events, nil
}
