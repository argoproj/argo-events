package webhook

import "github.com/argoproj/argo-events/gateways"

// StopConfig stops a configuration
func (wce *WebhookConfigExecutor) StopConfig(config *gateways.EventSourceContext) {
	if config.Active == true {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}
