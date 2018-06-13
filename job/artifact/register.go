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

package artifact

import (
	"fmt"

	"github.com/blackrock/axis/job"
	"github.com/blackrock/axis/job/nats"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	"go.uber.org/zap"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) (job.Signal, error) {
	abstract.Log.Info("creating signal", zap.String("raw", abstract.Artifact.String()))
	streamSignal := v1alpha1.Signal{
		Stream: &abstract.Artifact.NotificationStream,
	}
	var streamFactory job.Factory
	var ok bool
	switch abstract.Artifact.NotificationStream.Type {
	case nats.StreamTypeNats:
		streamFactory, ok = abstract.Session.GetStreamFactory(nats.StreamTypeNats)
		if !ok {
			abstract.Log.Warn("failed to initialize NATS stream registry properly, attempting manual add")
			nats.NATS(abstract.Session)
			streamFactory, ok = abstract.Session.GetStreamFactory(nats.StreamTypeNats)
		}
	default:
		return nil, fmt.Errorf("unknown stream type for artifact signal")
	}
	if !ok {
		return nil, fmt.Errorf("failed to find and/or initialize stream registry")
	}
	streamAbstract := job.AbstractSignal{
		Signal:  streamSignal,
		Log:     abstract.Log,
		Session: abstract.Session,
	}
	realStreamSignal, err := streamFactory.Create(streamAbstract)
	if err != nil {
		return nil, err
	}
	return &artifact{
		AbstractSignal: abstract,
		streamSignal:   realStreamSignal,
	}, nil
}

// Artifact will be added to the executor session
func Artifact(es *job.ExecutorSession) {
	es.AddCoreFactory(v1alpha1.SignalTypeArtifact, &factory{})
}
