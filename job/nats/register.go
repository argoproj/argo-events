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

package nats

import (
	"github.com/blackrock/axis/job"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	natsio "github.com/nats-io/go-nats"
	"go.uber.org/zap"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) job.Signal {
	abstract.Log.Info("creating signal", zap.String("raw", abstract.NATS.String()))
	return &nats{
		AbstractSignal: abstract,
		stop:           make(chan struct{}),
		msgCh:          make(chan *natsio.Msg),
	}
}

// NATS will be added to the executor session
func NATS(es *job.ExecutorSession) {
	es.AddFactory(v1alpha1.SignalTypeNats, &factory{})
}
