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
func (ese *WebhookEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	w, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}

	if w == nil {
		return v, fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidEventSource)
	}

	switch w.Method {
	case http.MethodHead, http.MethodPut, http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodTrace:
	default:
		return v, fmt.Errorf("%+v, unknown HTTP method %s", gateways.ErrInvalidEventSource, w.Method)
	}

	if w.Endpoint == "" {
		return v, fmt.Errorf("%+v, endpoint can't be empty", gateways.ErrInvalidEventSource)
	}
	if w.Port == "" {
		return v, fmt.Errorf("%+v, port can't be empty", gateways.ErrInvalidEventSource)
	}

	if !strings.HasPrefix(w.Endpoint, "/") {
		return v, fmt.Errorf("%+v, endpoint must start with '/'", gateways.ErrInvalidEventSource)
	}

	if w.Port != "" {
		_, err := strconv.Atoi(w.Port)
		if err != nil {
			return v, fmt.Errorf("%+v, failed to parse server port %s. err: %+v", gateways.ErrInvalidEventSource, w.Port, err)
		}
	}
	return v, nil
}
