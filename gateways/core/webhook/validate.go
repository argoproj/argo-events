package webhook

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	"net/http"
	"strconv"
	"strings"
)

// Validate validates given webhook configuration
func (wce *WebhookConfigExecutor) Validate(config *gateways.EventSourceContext) error {
	w, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}

	if w == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}

	switch w.Method {
	case http.MethodHead, http.MethodPut, http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodTrace:
	default:
		return fmt.Errorf("%+v, unknown HTTP method %s", gateways.ErrInvalidConfig, w.Method)
	}

	if w.Endpoint == "" {
		return fmt.Errorf("%+v, endpoint can't be empty", gateways.ErrInvalidConfig)
	}

	if !strings.HasPrefix(w.Endpoint, "/") {
		return fmt.Errorf("%+v, endpoint must start with '/'", gateways.ErrInvalidConfig)
	}

	if w.Port != "" {
		_, err := strconv.Atoi(w.Port)
		if err != nil {
			return fmt.Errorf("%+v, failed to parse server port %s. err: %+v", gateways.ErrInvalidConfig, w.Port, err)
		}
	}

	return nil
}
