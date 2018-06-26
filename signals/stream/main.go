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
	"github.com/argoproj/argo-events/shared"
	"github.com/argoproj/argo-events/signals/stream/builtin/nats"
	plugin "github.com/hashicorp/go-plugin"
)

func main() {
	// these are the stream plugin signals
	nats := nats.New()
	/*
		mqtt := mqtt.New()
		kafka := kafka.New()
		amqp := amqp.New()
	*/

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			"NATS": shared.NewPlugin(nats),
			//"MQTT":     shared.NewPlugin(mqtt),
			//"KAFKA":    shared.NewPlugin(kafka),
			//"AMQP":     shared.NewPlugin(amqp),
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
