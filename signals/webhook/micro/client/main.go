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
	"io"
	"net/http"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	"github.com/argoproj/argo-events/shared"
	log "github.com/sirupsen/logrus"
)

func main() {
	webhook := shared.NewMicroSignalClient().NewSignalService("webhook")

	signal := &v1alpha1.Signal{
		Name: "webhook-1",
		Webhook: &v1alpha1.WebhookSignal{
			Endpoint: "/hello",
			Method:   "POST",
		},
	}

	stream, err := webhook.Listen(context.Background(), signal)
	if err != nil {
		log.Panicf("failed to listen to webhook: %s", err)
	}

	waitc := make(chan struct{})
	go func() {
		defer close(waitc)
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Panicf("error during processing: %s", err)
			}
			log.Printf("received event: %v", *event)
		}
	}()

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://webhook:7070/hello", bytes.NewBuffer([]byte(`{"message":"Buy cheese and bread for breakfast"}`)))
	_, err = client.Do(req)
	if err != nil {
		log.Panicf("failed to post http: %s", err)
	}

	// now terminate the signal
	err = stream.Send(sdk.Terminate)
	if err != nil {
		log.Printf("failed to send stop signal: %s", err)
	}

	err = stream.Close()
	if err != nil {
		log.Printf("failed to close stream: %s", err)
	}
	<-waitc
	log.Printf("exiting signal client")
}
