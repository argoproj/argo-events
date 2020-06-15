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

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CustomTrigger implements Trigger interface for custom trigger resource
type CustomTrigger struct {
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger definition
	Trigger *v1alpha1.Trigger
	// logger to log stuff
	Logger *logrus.Entry
	// triggerClient is the gRPC client for the custom trigger server
	triggerClient triggers.TriggerClient
}

// NewCustomTrigger returns a new custom trigger
func NewCustomTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger, customTriggerClients map[string]*grpc.ClientConn) (*CustomTrigger, error) {
	customTrigger := &CustomTrigger{
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger.WithField("trigger-name", trigger.Template.Name),
	}

	ct := trigger.Template.CustomTrigger

	if conn, ok := customTriggerClients[trigger.Template.Name]; ok {
		if conn.GetState() == connectivity.Ready {
			logger.Infoln("trigger client connection is ready...")
			customTrigger.triggerClient = triggers.NewTriggerClient(conn)
			return customTrigger, nil
		}

		logger.Infoln("trigger client connection is closed, creating new one...")
		delete(customTriggerClients, trigger.Template.Name)
	}

	logger.WithField("server-url", ct.ServerURL).Infoln("instantiating trigger client...")

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

	logger.Infoln("successfully setup the trigger client...")
	return customTrigger, nil
}

// FetchResource fetches the trigger resource from external source
func (ct *CustomTrigger) FetchResource() (interface{}, error) {
	specBody, err := yaml.Marshal(ct.Trigger.Template.CustomTrigger.Spec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse the custom trigger spec body")
	}

	ct.Logger.WithField("spec", string(specBody)).Debugln("trigger spec body")

	resource, err := ct.triggerClient.FetchResource(context.Background(), &triggers.FetchResourceRequest{
		Resource: specBody,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch the custom trigger resource for %s", ct.Trigger.Template.Name)
	}

	ct.Logger.WithField("resource", string(resource.Resource)).Debugln("fetched resource")
	return resource.Resource, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (ct *CustomTrigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
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

		result, err := triggers.ApplyParams(obj, ct.Trigger.Template.CustomTrigger.Parameters, triggers.ExtractEvents(sensor, parameters))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply the parameters to the custom trigger resource for %s", ct.Trigger.Template.Name)
		}

		ct.Logger.WithField("resource", string(result)).Debugln("resource after parameterization")
		return result, nil
	}

	return resource, nil
}

// Execute executes the trigger
func (ct *CustomTrigger) Execute(resource interface{}) (interface{}, error) {
	obj, ok := resource.([]byte)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource for the execution")
	}

	ct.Logger.WithField("resource", string(obj)).Debugln("resource to execute")

	trigger := ct.Trigger.Template.CustomTrigger

	var payload []byte
	var err error

	if trigger.Payload != nil {
		payload, err = triggers.ConstructPayload(ct.Sensor, trigger.Payload)
		if err != nil {
			return nil, err
		}

		ct.Logger.WithField("payload", string(payload)).Debugln("payload for the trigger execution")
	}

	result, err := ct.triggerClient.Execute(context.Background(), &triggers.ExecuteRequest{
		Resource: obj,
		Payload:  payload,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute the custom trigger resource for %s", ct.Trigger.Template.Name)
	}

	ct.Logger.WithField("response", string(result.Response)).Debugln("trigger execution response")
	return result.Response, nil
}

// ApplyPolicy applies the policy on the trigger
func (ct *CustomTrigger) ApplyPolicy(resource interface{}) error {
	obj, ok := resource.([]byte)
	if !ok {
		return errors.New("failed to interpret the trigger resource for the policy application")
	}

	ct.Logger.WithField("resource", string(obj)).Debugln("resource to apply policy on")

	result, err := ct.triggerClient.ApplyPolicy(context.Background(), &triggers.ApplyPolicyRequest{
		Request: obj,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to apply the policy for the custom trigger resource for %s", ct.Trigger.Template.Name)
	}
	ct.Logger.WithFields(logrus.Fields{
		"success": result.Success,
		"message": result.Message,
	}).Infoln("policy application result")
	return err
}
