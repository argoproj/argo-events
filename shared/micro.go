package shared

import (
	"github.com/argoproj/argo-events/sdk"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	k8s "github.com/micro/kubernetes/go/micro"
)

// MicroSignalClient is the micro Client()
// NOTE: we can't use a default "static" client because tests use this package
// and will fail with: /var/run/secrets/kubernetes.io/serviceaccount: no such file or directory
type MicroSignalClient struct {
	client client.Client
}

/*
func init() {
	svc := k8s.NewService(micro.Name("signal.client"))
	svc.Init()
	defaultMicroSignalService = microSignalClient{client: svc.Client()}
}

var defaultMicroSignalService microSignalClient

// DefaultMicroSignalClient is the default MicroSignalClient
var DefaultMicroSignalClient = &defaultMicroSignalService
*/

// NewMicroSignalClient returns a new Micro Signal Client
func NewMicroSignalClient() *MicroSignalClient {
	svc := k8s.NewService(micro.Name("signal.client"))
	svc.Init()
	return &MicroSignalClient{client: svc.Client()}
}

// NewSignalService creates a new SignalService for the following service name
func (m *MicroSignalClient) NewSignalService(name string) sdk.SignalClient {
	return sdk.NewMicroSignalClient(name, m.client)
}

// DiscoverSignals returns all the registered signal microservices
// TODO: add svc Debug.Health check
func (m *MicroSignalClient) DiscoverSignals() ([]string, error) {
	svcs, err := m.client.Options().Registry.ListServices()
	if err != nil {
		return nil, err
	}
	svcNames := make([]string, 0)
	for _, svc := range svcs {
		if val := svc.Metadata[sdk.MetadataKey]; val == sdk.MetadataValue {
			svcNames = append(svcNames, svc.Name)
		}
	}
	return svcNames, nil
}
