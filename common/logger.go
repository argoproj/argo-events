package common

import "github.com/rs/zerolog"

type ArgoEventsLogger struct {
	*zerolog.Logger
}

func NewArgoEventsLogger() *ArgoEventsLogger {
	logger := GetLoggerContext(LoggerConf()).Logger()
	return &ArgoEventsLogger{
		&logger,
	}
}

func (a *ArgoEventsLogger) WithNamespace(namespace string) *ArgoEventsLogger {
	a.With().Str(LabelNamespace, namespace)
	return a
}

func (a *ArgoEventsLogger) WithGatewayControllerName(name string) *ArgoEventsLogger {
	a.With().Str(LabelGatewayControllerName, name)
	return a
}

func (a *ArgoEventsLogger) WithGatewayName(name string) *ArgoEventsLogger {
	a.With().Str(LabelGatewayName, name)
	return a
}

func (a *ArgoEventsLogger) WithSensorControllerName(name string) *ArgoEventsLogger {
	a.With().Str(LabelSensorControllerName, name)
	return a
}

func (a *ArgoEventsLogger) WithSensorName(name string) *ArgoEventsLogger {
	a.With().Str(LabelSensorName, name)
	return a
}

func (a *ArgoEventsLogger) WithPhase(phase string) *ArgoEventsLogger {
	a.With().Str(LabelPhase, phase)
	return a
}

func (a *ArgoEventsLogger) WithInstanceId(instanceId string) *ArgoEventsLogger {
	a.With().Str(LabelInstanceId, instanceId)
	return a
}

func (a *ArgoEventsLogger) WithPodName(podName string) *ArgoEventsLogger {
	a.With().Str(LabelPodName, podName)
	return a
}

func (a *ArgoEventsLogger) WithServiceName(svcName string) *ArgoEventsLogger {
	a.With().Str(LabelServiceName, svcName)
	return a
}

func (a *ArgoEventsLogger) WithEventSource(name string) *ArgoEventsLogger {
	a.With().Str(LabelEventSource, name)
	return a
}

func (a *ArgoEventsLogger) WithEndpoint(endpoint string) *ArgoEventsLogger {
	a.With().Str(LabelEndpoint, endpoint)
	return a
}

func (a *ArgoEventsLogger) WithPort(port string) *ArgoEventsLogger {
	a.With().Str(LabelPort, port)
	return a
}

func (a *ArgoEventsLogger) WithURL(url string) *ArgoEventsLogger {
	a.With().Str(LabelURL, url)
	return a
}

func (a *ArgoEventsLogger) WithVersion(version string) *ArgoEventsLogger {
	a.With().Str(LabelVersion, version)
	return a
}

func (a *ArgoEventsLogger) WithHttpMethod(method string) *ArgoEventsLogger {
	a.With().Str(LabelHttpMethod, method)
	return a
}
