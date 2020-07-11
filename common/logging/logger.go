/*
Copyright 2020 BlackRock, Inc.

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

package logging

import (
	"context"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"

	"github.com/argoproj/argo-events/common"
)

// Logger constants
const (
	LabelEventSourceType = "eventSourceType"
	LabelEventSourceName = "eventSourceName"
	LabelEventName       = "eventName"
	LabelNamespace       = "namespace"
	LabelPhase           = "phase"
	LabelInstanceID      = "instance-id"
	LabelEndpoint        = "endpoint"
	LabelPort            = "port"
	LabelHTTPMethod      = "http-method"
	LabelVersion         = "version"
	LabelTime            = "time"
	TimestampFormat      = "2006-01-02 15:04:05"
)

// NewArgoEventsLogger returns a new ArgoEventsLogger
func NewArgoEventsLogger() *logrus.Logger {
	var formatter logrus.Formatter

	// If out is term use textformatter
	if isatty.IsTerminal(os.Stdout.Fd()) {
		formatter = &logrus.TextFormatter{
			TimestampFormat:  TimestampFormat,
			FullTimestamp:    true,
			ForceColors:      true,
			QuoteEmptyFields: true,
		}
	} else {
		formatter = &logrus.JSONFormatter{
			TimestampFormat: TimestampFormat,
		}
	}

	log := &logrus.Logger{
		Out:       os.Stdout,
		Level:     logrus.InfoLevel,
		Formatter: formatter,
	}

	debugMode, ok := os.LookupEnv(common.EnvVarDebugLog)
	if ok && debugMode == "true" {
		log.SetLevel(logrus.DebugLevel)
	}

	return log
}

type loggerKey struct{}

// WithLogger returns a copy of parent context in which the
// value associated with logger key is the supplied logger.
func WithLogger(ctx context.Context, logger *logrus.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext returns the logger in the context.
func FromContext(ctx context.Context) *logrus.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*logrus.Logger); ok {
		return logger
	}
	return NewArgoEventsLogger()
}
