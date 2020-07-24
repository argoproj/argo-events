/*
Copyright 2020 BlackRock, Inc.

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
package customtrigger

import (
	"context"
	"encoding/json"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
)

// CustomTrigger implements Trigger interface for custom trigger resource
type CustomTrigger struct {
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger definition
	Trigger *v1alpha1.Trigger
	// logger to log stuff
	Logger *zap.Logger
	// triggerClient is the gRPC client for the custom trigger server
	triggerClient triggers.TriggerClient
}

// NewCustomTrigger returns a new custom trigger
func NewCustomTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.Logger, customTriggerClients map[string]*grpc.ClientConn) (*CustomTrigger, error) {
	customTrigger := &CustomTrigger{
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger.Sugar().With("trigger-name", trigger.Template.Name).Desugar(),
	}

	ct := trigger.Template.CustomTrigger

	if conn, ok := customTriggerClients[trigger.Template.Name]; ok {
		if conn.GetState() == connectivity.Ready {
			logger.Info("trigger client connection is ready...")
			customTrigger.triggerClient = triggers.NewTriggerClient(conn)
			return customTrigger, nil
		}

		logger.Info("trigger client connection is closed, creating new one...")
		delete(customTriggerClients, trigger.Template.Name)
	}

	logger.Info("instantiating trigger client...", zap.Any("server-url", ct.ServerURL))

	opt := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithInsecure(),
	}

	if ct.Secure {
		creds, err := credentials.NewClientTLSFromFile(ct.CertFilePath, ct.ServerNameOverride)
		if err != nil {
			return nil, err
		}
		opt = append(opt, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.Dial(
		ct.ServerURL,
		opt...,
	)
	if err != nil {
		return nil, err
	}

	connBackoff := common.GetConnectionBackoff(nil)

	if err = wait.ExponentialBackoff(*connBackoff, func() (done bool, err error) {
		if conn.GetState() == connectivity.Ready {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return nil, err
	}

	customTrigger.triggerClient = triggers.NewTriggerClient(conn)
	customTriggerClients[trigger.Template.Name] = conn

	logger.Info("successfully setup the trigger client...")
	return customTrigger, nil
}

// FetchResource fetches the trigger resource from external source
func (ct *CustomTrigger) FetchResource() (interface{}, error) {
	specBody, err := yaml.Marshal(ct.Trigger.Template.CustomTrigger.Spec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse the custom trigger spec body")
	}

	ct.Logger.Debug("trigger spec body", zap.Any("spec", string(specBody)))

	resource, err := ct.triggerClient.FetchResource(context.Background(), &triggers.FetchResourceRequest{
		Resource: specBody,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch the custom trigger resource for %s", ct.Trigger.Template.Name)
	}

	ct.Logger.Debug("fetched resource", zap.Any("resource", string(resource.Resource)))
	return resource.Resource, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (ct *CustomTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	obj, ok := resource.([]byte)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource for resource parameters application")
	}
	parameters := ct.Trigger.Template.CustomTrigger.Parameters

	if len(parameters) > 0 {
		// only JSON formatted resource body is eligible for parameters
		var temp map[string]interface{}
		if err := json.Unmarshal(obj, &temp); err != nil {
			return nil, errors.Wrapf(err, "fetched resource body is not valid JSON for trigger %s", ct.Trigger.Template.Name)
		}

		result, err := triggers.ApplyParams(obj, ct.Trigger.Template.CustomTrigger.Parameters, events)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply the parameters to the custom trigger resource for %s", ct.Trigger.Template.Name)
		}

		ct.Logger.Debug("resource after parameterization", zap.Any("resource", string(result)))
		return result, nil
	}

	return resource, nil
}

// Execute executes the trigger
func (ct *CustomTrigger) Execute(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	obj, ok := resource.([]byte)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource for the execution")
	}

	ct.Logger.Debug("resource to execute", zap.Any("resource", string(obj)))

	trigger := ct.Trigger.Template.CustomTrigger

	var payload []byte
	var err error

	if trigger.Payload != nil {
		payload, err = triggers.ConstructPayload(events, trigger.Payload)
		if err != nil {
			return nil, err
		}

		ct.Logger.Debug("payload for the trigger execution", zap.Any("payload", string(payload)))
	}

	result, err := ct.triggerClient.Execute(context.Background(), &triggers.ExecuteRequest{
		Resource: obj,
		Payload:  payload,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute the custom trigger resource for %s", ct.Trigger.Template.Name)
	}

	ct.Logger.Debug("trigger execution response", zap.Any("response", string(result.Response)))
	return result.Response, nil
}

// ApplyPolicy applies the policy on the trigger
func (ct *CustomTrigger) ApplyPolicy(resource interface{}) error {
	obj, ok := resource.([]byte)
	if !ok {
		return errors.New("failed to interpret the trigger resource for the policy application")
	}

	ct.Logger.Debug("resource to apply policy on", zap.Any("resource", string(obj)))

	result, err := ct.triggerClient.ApplyPolicy(context.Background(), &triggers.ApplyPolicyRequest{
		Request: obj,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to apply the policy for the custom trigger resource for %s", ct.Trigger.Template.Name)
	}
	ct.Logger.Info("policy application result", zap.Any("success", result.Success), zap.Any("message", result.Message))
	return err
}
