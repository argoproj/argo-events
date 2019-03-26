package common

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger constants
const (
	LabelNamespace   = "namespace"
	LabelPhase       = "phase"
	LabelInstanceId  = "instance-id"
	LabelPodName     = "pod-name"
	LabelServiceName = "svc-name"
	LabelEndpoint    = "endpoint"
	LabelPort        = "port"
	LabelURL         = "url"
	LabelNodeName    = "node-name"
	LabelNodeType    = "node-type"
	LabelHttpMethod  = "http-method"
	LabelClientId    = "client-id"
	LabelVersion     = "version"
	LabelError       = "error"
	LabelTime        = "time"
)

type ArgoEventsLogger struct {
	*logrus.Logger
}

func NewArgoEventsLogger() *ArgoEventsLogger {
	logrus.SetOutput(os.Stdout)
	return &ArgoEventsLogger{
		logrus.New(),
	}
}

func (a *ArgoEventsLogger) WithNamespace(namespace string) *ArgoEventsLogger {
	a.WithField(LabelNamespace, namespace)
	return a
}

func (a *ArgoEventsLogger) WithGatewayControllerName(name string) *ArgoEventsLogger {
	a.WithField(LabelGatewayControllerName, name)
	return a
}

func (a *ArgoEventsLogger) WithGatewayName(name string) *ArgoEventsLogger {
	a.WithField(LabelGatewayName, name)
	return a
}

func (a *ArgoEventsLogger) WithSensorControllerName(name string) *ArgoEventsLogger {
	a.WithField(LabelSensorControllerName, name)
	return a
}

func (a *ArgoEventsLogger) WithSensorName(name string) *ArgoEventsLogger {
	a.WithField(LabelSensorName, name)
	return a
}

func (a *ArgoEventsLogger) WithPhase(phase string) *ArgoEventsLogger {
	a.WithField(LabelPhase, phase)
	return a
}

func (a *ArgoEventsLogger) WithInstanceId(instanceId string) *ArgoEventsLogger {
	a.WithField(LabelInstanceId, instanceId)
	return a
}

func (a *ArgoEventsLogger) WithPodName(podName string) *ArgoEventsLogger {
	a.WithField(LabelPodName, podName)
	return a
}

func (a *ArgoEventsLogger) WithServiceName(svcName string) *ArgoEventsLogger {
	a.WithField(LabelServiceName, svcName)
	return a
}

func (a *ArgoEventsLogger) WithEventSource(name string) *ArgoEventsLogger {
	a.WithField(LabelEventSource, name)
	return a
}

func (a *ArgoEventsLogger) WithEndpoint(endpoint string) *ArgoEventsLogger {
	a.WithField(LabelEndpoint, endpoint)
	return a
}

func (a *ArgoEventsLogger) WithPort(port string) *ArgoEventsLogger {
	a.WithField(LabelPort, port)
	return a
}

func (a *ArgoEventsLogger) WithURL(url string) *ArgoEventsLogger {
	a.WithField(LabelURL, url)
	return a
}

func (a *ArgoEventsLogger) WithVersion(version string) *ArgoEventsLogger {
	a.WithField(LabelVersion, version)
	return a
}

func (a *ArgoEventsLogger) WithHttpMethod(method string) *ArgoEventsLogger {
	a.WithField(LabelHttpMethod, method)
	return a
}

func (a *ArgoEventsLogger) WithError(err error) *ArgoEventsLogger {
	a.WithField(LabelError, err)
	return a
}

func (a *ArgoEventsLogger) WithTime(time string) *ArgoEventsLogger {
	a.WithField(LabelTime, time)
	return a
}
