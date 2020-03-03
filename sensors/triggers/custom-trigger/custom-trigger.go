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
package custom_trigger

import (
	"context"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
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
	Logger *logrus.Logger
	// triggerClient is the gRPC client for the custom trigger server
	triggerClient triggers.TriggerClient
}

func NewCustomTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger, customTriggerClients map[string]*grpc.ClientConn) (*CustomTrigger, error) {
	customTrigger := &CustomTrigger{
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger,
	}

	ct := trigger.Template.CustomTrigger

	if conn, ok := customTriggerClients[trigger.Template.Name]; ok {
		if conn.GetState() == connectivity.Ready {
			customTrigger.triggerClient = triggers.NewTriggerClient(conn)
			return customTrigger, nil
		}
		delete(customTriggerClients, trigger.Template.Name)
	}

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
	return customTrigger, nil
}

// FetchResource fetches the trigger resource from external source
func (ct *CustomTrigger) FetchResource() (interface{}, error) {
	resource, err := ct.triggerClient.FetchResource(context.Background(), &triggers.FetchResourceRequest{
		Resource: []byte(ct.Trigger.Template.CustomTrigger.TriggerBody),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch the custom trigger resource for %s", ct.Trigger.Template.Name)
	}
	return resource, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (ct *CustomTrigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
	obj, ok := resource.([]byte)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource for resource parameters application")
	}
	parameters := ct.Trigger.Template.CustomTrigger.Parameters

	if parameters != nil && len(parameters) > 0 {
		resource, err := triggers.ApplyParams(obj, ct.Trigger.Template.OpenFaas.Parameters, triggers.ExtractEvents(sensor, parameters))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply the parameters to the custom trigger resource for %s", ct.Trigger.Template.Name)
		}
		return resource, nil
	}

	return resource, nil
}

// Execute executes the trigger
func (ct *CustomTrigger) Execute(resource interface{}) (interface{}, error) {
	obj, ok := resource.([]byte)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource for the execution")
	}
	trigger := ct.Trigger.Template.CustomTrigger
	if trigger.Payload == nil {
		return nil, errors.New("payload parameters are not specified")
	}
	payload, err := triggers.ConstructPayload(ct.Sensor, trigger.Payload)
	if err != nil {
		return nil, err
	}
	result, err := ct.triggerClient.Execute(context.Background(), &triggers.ExecuteRequest{
		Resource: obj,
		Payload:  payload,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute the custom trigger resource for %s", ct.Trigger.Template.Name)
	}
	return result, nil
}

// ApplyPolicy applies the policy on the trigger
func (ct *CustomTrigger) ApplyPolicy(resource interface{}) error {
	obj, ok := resource.([]byte)
	if !ok {
		return errors.New("failed to interpret the trigger resource for the policy application")
	}
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
