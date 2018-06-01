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
	"net/http"
	"go.uber.org/zap"

	"github.com/blackrock/axis/job"
	"io"
	"strconv"
	"fmt"
	"time"
)

type webhook struct {
	job.AbstractSignal
	events chan job.Event
	server *http.Server
	payload io.ReadCloser
}

// Handler for the http rest endpoint
func (w *webhook) handler(writer http.ResponseWriter, request *http.Request) {
	w.Log.Info("received a request from", zap.String("host", request.Host))
	if request.Method == w.Webhook.Method {
		w.payload = request.Body
		event := &event{
			webhook: w,
			requestHost: request.Host,
			timestamp: time.Now().UTC(),
		}
		w.events <- event
		writer.WriteHeader(http.StatusOK)
	} else {
		w.Log.Warn("HTTP method mismatch", zap.String("request method", request.Method), zap.String("signal method", w.Webhook.Method))
		writer.WriteHeader(http.StatusBadRequest)
	}
}

// Start signal
func (w *webhook) Start(events chan job.Event) error {
	w.events = events
	port := strconv.Itoa(w.AbstractSignal.Webhook.Port)
	endpoint := w.AbstractSignal.Webhook.Endpoint
	// Attach handler
	http.HandleFunc(endpoint, w.handler)
	w.server = &http.Server{Addr: fmt.Sprintf(":%s", port)}

	// Start http server
	go func() {
		err := w.server.ListenAndServe()
		if err == http.ErrServerClosed {
			w.Log.Info("server successfully shutdown")
		} else {
			panic(fmt.Errorf("error occurred while server listening. Cause: %v", err))
		}
	}()
	return nil
}

// Stop signal
func (w *webhook) Stop() error {
	err := w.server.Shutdown(nil)
	if err != nil {
		return fmt.Errorf("unable to shutdown server. Cause: %v", err)
	}
	return nil
}
