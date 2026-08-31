/*
Copyright 2026 The Argoproj Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/sensors/policy"
	"github.com/argoproj/argo-events/pkg/sensors/triggers"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

const defaultGRPCTimeout = 10 * time.Second

// GRPCTrigger describes the trigger to invoke an arbitrary unary gRPC method
type GRPCTrigger struct {
	// Conn is the gRPC client connection.
	Conn *grpc.ClientConn
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *zap.SugaredLogger
}

// NewGRPCTrigger returns a new GRPC trigger
func NewGRPCTrigger(grpcClients sharedutil.StringKeyedMap[*grpc.ClientConn], sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) (*GRPCTrigger, error) {
	log := logger.With(logging.LabelTriggerType, v1alpha1.TriggerTypeGRPC)
	grpcTrigger := trigger.Template.GRPC

	if conn, ok := grpcClients.Load(trigger.Template.Name); ok {
		return &GRPCTrigger{Conn: conn, Sensor: sensor, Trigger: trigger, Logger: log}, nil
	}

	var creds credentials.TransportCredentials
	switch {
	case grpcTrigger.Insecure:
		creds = insecure.NewCredentials()
	case grpcTrigger.TLS != nil:
		tlsConfig, err := sharedutil.GetTLSConfig(grpcTrigger.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		creds = credentials.NewTLS(tlsConfig)
	default:
		return nil, fmt.Errorf("either tls or insecure must be configured for the gRPC trigger")
	}

	conn, err := grpc.NewClient(grpcTrigger.URL, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the gRPC server, %w", err)
	}

	grpcClients.Store(trigger.Template.Name, conn)

	return &GRPCTrigger{Conn: conn, Sensor: sensor, Trigger: trigger, Logger: log}, nil
}

// GetTriggerType returns the type of the trigger
func (t *GRPCTrigger) GetTriggerType() v1alpha1.TriggerType {
	return v1alpha1.TriggerTypeGRPC
}

// FetchResource fetches the trigger. As the GRPC trigger simply invokes a
// gRPC method, there is no need to fetch any resource from an external source.
func (t *GRPCTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.GRPC, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *GRPCTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.GRPCTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the grpc trigger resource, %w", err)
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var gt *v1alpha1.GRPCTrigger
		if err := json.Unmarshal(updatedResourceBytes, &gt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated grpc trigger resource after applying resource parameters, %w", err)
		}
		return gt, nil
	}
	return resource, nil
}

// Execute executes the trigger. The response body is never decoded — the
// returned value is always a *status.Status, even when the RPC completes
// with a non-OK code, mirroring how HTTPTrigger.Execute returns the raw
// *http.Response for non-2xx codes. A non-nil error here means the call
// could not even be attempted (bad schema, malformed payload).
func (t *GRPCTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.GRPCTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the trigger resource")
	}

	desc, err := buildRequestDescriptor(trigger.Schema)
	if err != nil {
		return nil, err
	}

	var payload []byte
	if trigger.Payload != nil {
		payload, err = triggers.ConstructPayload(events, trigger.Payload)
		if err != nil {
			return nil, err
		}
	}

	msg := dynamicpb.NewMessage(desc)
	if len(payload) > 0 {
		if err := protojson.Unmarshal(payload, msg); err != nil {
			return nil, fmt.Errorf("failed to construct the gRPC request message, %w", err)
		}
	}

	timeout := defaultGRPCTimeout
	if trigger.Timeout > 0 {
		timeout = time.Duration(trigger.Timeout) * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	t.Logger.Infow("invoking gRPC method...", zap.Any("url", trigger.URL), zap.Any("method", trigger.Method))

	reply := dynamicpb.NewMessage(emptyResponseDescriptor)
	invokeErr := t.Conn.Invoke(callCtx, trigger.Method, msg, reply)
	return status.Convert(invokeErr), nil
}

// ApplyPolicy applies policy on the trigger, reusing the same generic
// status-code allow-list mechanism every other trigger type uses
// (v1alpha1.Trigger.Policy.Status.Allow), interpreting Allow's values as
// gRPC status codes instead of HTTP status codes.
func (t *GRPCTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	if t.Trigger.Policy == nil || t.Trigger.Policy.Status == nil || t.Trigger.Policy.Status.Allow == nil {
		return nil
	}
	st, ok := resource.(*status.Status)
	if !ok {
		return fmt.Errorf("failed to interpret the trigger execution response")
	}

	p := policy.NewStatusPolicy(int(st.Code()), t.Trigger.Policy.Status.GetAllow())
	return p.ApplyPolicy(ctx)
}
