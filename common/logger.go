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

package common

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

// Logger constants
const (
	LabelNamespace   = "namespace"
	LabelPhase       = "phase"
	LabelInstanceID  = "instance-id"
	LabelPodName     = "pod-name"
	LabelServiceName = "svc-name"
	LabelEndpoint    = "endpoint"
	LabelPort        = "port"
	LabelURL         = "url"
	LabelNodeName    = "node-name"
	LabelNodeType    = "node-type"
	LabelHTTPMethod  = "http-method"
	LabelClientID    = "client-id"
	LabelVersion     = "version"
	LabelTime        = "time"
	LabelTriggerName = "trigger-name"
)

// NewArgoEventsLogger returns a new ArgoEventsLogger
func NewArgoEventsLogger() *logrus.Logger {
	log := logrus.New()

	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	debugMode, ok := os.LookupEnv(EnvVarDebugLog)
	if ok && debugMode == "true" {
		log.SetLevel(logrus.DebugLevel)
	}

	log.SetReportCaller(true)
	log.Formatter = &logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := strings.Split(f.File, "github.com/argoproj/argo-events")
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename[1], f.Line)
		},
	}

	return log
}
