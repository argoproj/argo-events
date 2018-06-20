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

package calendar

import (
	"sync"

	"github.com/argoproj/argo-events/job"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"go.uber.org/zap"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) (job.Signal, error) {
	abstract.Log.Info("creating signal", zap.String("raw", abstract.Calendar.String()))
	return &calendar{
		AbstractSignal: abstract,
		stop:           make(chan struct{}),
		wg:             sync.WaitGroup{},
	}, nil
}

// Calendar will be added to the executor session
func Calendar(es *job.ExecutorSession) {
	es.AddCoreFactory(v1alpha1.SignalTypeCalendar, &factory{})
}
