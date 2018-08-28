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
	"strconv"
	"testing"

	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/gnatsd/test"
	natsio "github.com/nats-io/go-nats"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"net/http"
	"io/ioutil"
	"github.com/argoproj/argo-events/common"
	"log"
)

const (
	subject = "mysubject"
	msg = "hello there"
)

func startHttpServer(t *testing.T) {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		m, err := ioutil.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		if msg != string(m) {
			t.Error("received message doesn't match sent message")
		}
	})
	log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%d", common.GatewayTransformerPort), nil))
}

func TestSignal(t *testing.T) {
	startHttpServer(t)
	natsEmbeddedServerOpts := server.Options{
		Host:           "localhost",
		Port:           4222,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 256,
	}
	natsServer := test.RunServer(&natsEmbeddedServerOpts)
	defer natsServer.Shutdown()

	natsConfig := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nats-gateway-config",
		},
		Data: map[string]string{
			"nats-config": fmt.Sprintf(`|-
							url: %s
							attributes:
								subject: %s`, "nats://" + natsEmbeddedServerOpts.Host + ":" + strconv.Itoa(natsEmbeddedServerOpts.Port), subject),
		},
	}

	n := &nats{
		registeredNATS: make(map[uint64]*stream.Stream),
	}
	err := n.RunGateway(natsConfig)

	if err != nil {
		t.Error(err)
	}

	// publish a message
	conn, err := natsio.Connect("nats://" + natsEmbeddedServerOpts.Host + ":" + strconv.Itoa(natsEmbeddedServerOpts.Port))
	if err != nil {
		t.Fatalf("failed to connect to embedded nats server. cause: %s", err)
	}
	defer conn.Close()
	err = conn.Publish("mysubject", []byte(msg))
	if err != nil {
		t.Fatalf("failed to publish test msg. cause: %s", err)
	}
}
