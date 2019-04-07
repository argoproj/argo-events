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
	"os"
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
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
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	debugMode, ok := os.LookupEnv(EnvVarDebugLog)
	if ok && debugMode == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.AddHook(&ContextHook{})

	return logrus.StandardLogger()
}

type ContextHook struct{}

func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook ContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(10); ok {
		funcName := runtime.FuncForPC(pc).Name()
		entry.Data["source"] = fmt.Sprintf("%s:%v:%s", path.Base(file), line, path.Base(funcName))
	}
	return nil
}
