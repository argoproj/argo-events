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
	"fmt"

	"github.com/argoproj/argo-events/job"
	natsio "github.com/nats-io/go-nats"
	"go.uber.org/zap"
)

const (
	// StreamTypeNats defines the exact string representation for NATS stream types
	StreamTypeNats = "NATS"
	subjectKey     = "subject"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) (job.Signal, error) {
	abstract.Log.Info("creating signal", zap.String("type", abstract.Stream.Type))
	n := &nats{
		AbstractSignal: abstract,
		stop:           make(chan struct{}),
		msgCh:          make(chan *natsio.Msg),
	}
	var ok bool
	if n.subject, ok = abstract.Stream.Attributes[subjectKey]; !ok {
		return nil, fmt.Errorf(job.ErrMissingAttribute, abstract.Stream.Type, subjectKey)
	}

	return n, nil
}

// NATS will be added to the executor session
func NATS(es *job.ExecutorSession) {
	es.AddStreamFactory(StreamTypeNats, &factory{})
}
