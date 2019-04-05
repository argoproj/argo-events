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
	"os"

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
	LabelError       = "error"
	LabelTime        = "time"
	LabelTriggerName = "trigger-name"
)

// ArgoEventsLogger is the logger for the framework
type ArgoEventsLogger struct {
	*logrus.Logger
}

// NewArgoEventsLogger returns a new ArgoEventsLogger
func NewArgoEventsLogger() *ArgoEventsLogger {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	debugMode, ok := os.LookupEnv(EnvVarDebugLog)
	if ok && debugMode == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	return &ArgoEventsLogger{
		logrus.StandardLogger(),
	}
}

// WithNamespace returns logger with namespace field set
func (a *ArgoEventsLogger) WithNamespace(namespace string) *ArgoEventsLogger {
	a.WithField(LabelNamespace, namespace)
	return a
}

// WithGatewayControllerName returns logger with gateway controller name set
func (a *ArgoEventsLogger) WithGatewayControllerName(name string) *ArgoEventsLogger {
	a.WithField(LabelGatewayControllerName, name)
	return a
}

// WithGatewayName returns the logger with gateway name set
func (a *ArgoEventsLogger) WithGatewayName(name string) *ArgoEventsLogger {
	a.WithField(LabelGatewayName, name)
	return a
}

// WithSensorControllerName returns the logger with sensor controller name set
func (a *ArgoEventsLogger) WithSensorControllerName(name string) *ArgoEventsLogger {
	a.WithField(LabelSensorControllerName, name)
	return a
}

// WithSensorName returns the logger with sensor name set
func (a *ArgoEventsLogger) WithSensorName(name string) *ArgoEventsLogger {
	a.WithField(LabelSensorName, name)
	return a
}

// WithPhase returns the logger with phase set
func (a *ArgoEventsLogger) WithPhase(phase string) *ArgoEventsLogger {
	a.WithField(LabelPhase, phase)
	return a
}

// WithInstanceId returns the logger with instance id set
func (a *ArgoEventsLogger) WithInstanceId(instanceId string) *ArgoEventsLogger {
	a.WithField(LabelInstanceID, instanceId)
	return a
}

// WithPodName returns the logger with pod name set
func (a *ArgoEventsLogger) WithPodName(podName string) *ArgoEventsLogger {
	a.WithField(LabelPodName, podName)
	return a
}

// WithServiceName returns the logger with service name set
func (a *ArgoEventsLogger) WithServiceName(svcName string) *ArgoEventsLogger {
	a.WithField(LabelServiceName, svcName)
	return a
}

// WithEventSource returns the logger with event source set
func (a *ArgoEventsLogger) WithEventSource(name string) *ArgoEventsLogger {
	a.WithField(LabelEventSource, name)
	return a
}

// WithEndpoint returns the logger with endpoint set
func (a *ArgoEventsLogger) WithEndpoint(endpoint string) *ArgoEventsLogger {
	a.WithField(LabelEndpoint, endpoint)
	return a
}

// WithPort returns the logger with port set
func (a *ArgoEventsLogger) WithPort(port string) *ArgoEventsLogger {
	a.WithField(LabelPort, port)
	return a
}

// WithURL returns the logger with url set
func (a *ArgoEventsLogger) WithURL(url string) *ArgoEventsLogger {
	a.WithField(LabelURL, url)
	return a
}

// WithVersion returns the logger with version set
func (a *ArgoEventsLogger) WithVersion(version string) *ArgoEventsLogger {
	a.WithField(LabelVersion, version)
	return a
}

// WithHTTPMethod returns the logger with http method set
func (a *ArgoEventsLogger) WithHTTPMethod(method string) *ArgoEventsLogger {
	a.WithField(LabelHTTPMethod, method)
	return a
}

// WithError returns the logger with error set
func (a *ArgoEventsLogger) WithError(err error) *ArgoEventsLogger {
	a.WithField(LabelError, err)
	return a
}

// WithTime returns the logger with time set
func (a *ArgoEventsLogger) WithTime(time string) *ArgoEventsLogger {
	a.WithField(LabelTime, time)
	return a
}

// WithTrigger returns the logger with trigger name set
func (a *ArgoEventsLogger) WithTrigger(name string) *ArgoEventsLogger {
	a.WithField(LabelTriggerName, name)
	return a
}
