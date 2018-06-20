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
	"log"
	"net/http"
	"time"

	"fmt"
	"io/ioutil"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/shared"
)

const (
	EventType = "Webhook"
)

type webhook struct {
	method string
	events chan *v1alpha1.Event
	server *http.Server
}

// New creates a new webhook signaler
func New() shared.Signaler {
	return &webhook{}
}

// Handler for the http rest endpoint
func (w *webhook) handler(writer http.ResponseWriter, request *http.Request) {
	log.Printf("received a request from '%s'", request.Host)
	if request.Method == w.method {
		payload, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Printf("unable to process request payload. Cause: %s", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		event := &v1alpha1.Event{
			Context: v1alpha1.EventContext{
				EventType:          EventType,
				EventTypeVersion:   request.Proto,
				CloudEventsVersion: shared.CloudEventsVersion,
				Source: &v1alpha1.URI{
					Scheme: request.RequestURI,
					Host:   request.Host,
				},
				EventTime: metav1.Time{Time: time.Now().UTC()},
			},
			Data: payload,
		}
		w.events <- event
		writer.WriteHeader(http.StatusOK)
	} else {
		log.Printf("HTTP method of request '%s' does not match expected '%s'", request.Method, w.method)
		writer.WriteHeader(http.StatusBadRequest)
	}
}

// Start signal
func (w *webhook) Start(signal *v1alpha1.Signal) (<-chan *v1alpha1.Event, error) {
	w.method = signal.Webhook.Method
	port := strconv.Itoa(int(signal.Webhook.Port))
	endpoint := signal.Webhook.Endpoint
	// Attach handler
	http.HandleFunc(endpoint, w.handler)
	w.server = &http.Server{Addr: fmt.Sprintf(":%s", port)}

	w.events = make(chan *v1alpha1.Event)
	// Start http server
	go func() {
		err := w.server.ListenAndServe()
		if err == http.ErrServerClosed {
			log.Print("server successfully shutdown")
		} else {
			panic(fmt.Errorf("error occurred while server listening. Cause: %v", err))
		}
	}()
	return w.events, nil
}

// Stop signal
func (w *webhook) Stop() error {
	close(w.events)
	err := w.server.Shutdown(nil)
	if err != nil {
		return fmt.Errorf("unable to shutdown server. Cause: %v", err)
	}
	return nil
}
