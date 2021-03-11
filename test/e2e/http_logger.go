package e2e

import (
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common/logging"
)

type httpLogger struct {
	log *zap.SugaredLogger
}

func NewHttpLogger() *httpLogger {
	return &httpLogger{
		log: logging.NewArgoEventsLogger(),
	}
}

func (d *httpLogger) Logf(fmt string, args ...interface{}) {
	d.log.Debugf(fmt, args...)
}
