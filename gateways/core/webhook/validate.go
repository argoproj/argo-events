package webhook

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	"net/http"
	"strconv"
	"strings"
)

// ValidateEventSource validates webhook event source
func (wce *WebhookConfigExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	w, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrConfigParseFailed
	}

	if w == nil {
		return v, fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}

	switch w.Method {
	case http.MethodHead, http.MethodPut, http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodTrace:
	default:
		return v, fmt.Errorf("%+v, unknown HTTP method %s", gateways.ErrInvalidConfig, w.Method)
	}

	if w.Endpoint == "" {
		return v, fmt.Errorf("%+v, endpoint can't be empty", gateways.ErrInvalidConfig)
	}
	if w.Port == "" {
		return v, fmt.Errorf("%+v, port can't be empty", gateways.ErrInvalidConfig)
	}

	if !strings.HasPrefix(w.Endpoint, "/") {
		return v, fmt.Errorf("%+v, endpoint must start with '/'", gateways.ErrInvalidConfig)
	}

	if w.Port != "" {
		_, err := strconv.Atoi(w.Port)
		if err != nil {
			return v, fmt.Errorf("%+v, failed to parse server port %s. err: %+v", gateways.ErrInvalidConfig, w.Port, err)
		}
	}
	return v, nil
}
