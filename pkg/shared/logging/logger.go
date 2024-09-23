package logging

import (
	"context"
	"flag"
	"os"
	"strconv"

	zap "go.uber.org/zap"
	"k8s.io/klog/v2"
)

// Logger constants
const (
	LabelEventSourceType = "eventSourceType"
	LabelEventSourceName = "eventSourceName"
	LabelEventName       = "eventName"
	LabelTriggerName     = "triggerName"
	LabelTriggerType     = "triggerType"
	LabelNamespace       = "namespace"
	LabelEndpoint        = "endpoint"
	LabelPort            = "port"
	LabelHTTPMethod      = "http-method"
	LabelTime            = "time"
	TimestampFormat      = "2006-01-02 15:04:05"
	InfoLevel            = "info"
	DebugLevel           = "debug"
	ErrorLevel           = "error"
)

// NewArgoEventsLogger returns a new ArgoEventsLogger
func NewArgoEventsLogger() *zap.SugaredLogger {
	logLevel, _ := os.LookupEnv("LOG_LEVEL")
	config := ConfigureLogLevelLogger(logLevel)
	// Config customization goes here if any
	config.OutputPaths = []string{"stdout"}
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	return logger.Named("argo-events").Sugar()
}

func SetKlogLevel(level int) {
	klog.InitFlags(nil)
	_ = flag.Set("v", strconv.Itoa(level))
}

type loggerKey struct{}

// WithLogger returns a copy of parent context in which the
// value associated with logger key is the supplied logger.
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext returns the logger in the context.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerKey{}).(*zap.SugaredLogger); ok {
		return logger
	}
	return NewArgoEventsLogger()
}

// Returns logger conifg depending on the log level
func ConfigureLogLevelLogger(logLevel string) zap.Config {
	logConfig := zap.NewProductionConfig()
	switch logLevel {
	case InfoLevel:
		logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case ErrorLevel:
		logConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case DebugLevel:
		logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	default:
		logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	return logConfig
}
