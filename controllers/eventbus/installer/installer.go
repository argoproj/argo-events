package installer

import (
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// Installer is an interface for event bus installation
type Installer interface {
	Install() (*v1alpha1.BusConfig, error)
	Uninstall() error
}
