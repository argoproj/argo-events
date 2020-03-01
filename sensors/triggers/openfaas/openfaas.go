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
package openfaas

import (
	"encoding/json"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/argoproj/argo-events/sensors/types"
	openfaas "github.com/openfaas-incubator/connector-sdk/types"
	"github.com/openfaas/faas-provider/auth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// OpenFaasTrigger holds the context to invoke OpenFaas functions
type OpenFaasTrigger struct {
	// K8sClient is the Kubernetes client
	K8sClient kubernetes.Interface
	// Sensor refer to the sensor object
	Sensor *v1alpha1.Sensor
	// Trigger refers to the trigger resource
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *logrus.Logger
	// Controller is the openfaas connector controller
	OpenFaasContext *types.OpenFaasContext
}

// ResponseReceiver enables connector to receive results from the
// function invocation
type ResponseReceiver struct {
	ResponseCh chan openfaas.InvokerResponse
}

// Response is triggered by the controller when a message is
// received from the function invocation
func (r ResponseReceiver) Response(res openfaas.InvokerResponse) {
	r.ResponseCh <- res
}

// NewOpenFaasTrigger returns a new OpenFaas trigger context
func NewOpenFaasTrigger(k8sClient kubernetes.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger, connectors map[string]*types.OpenFaasContext) (*OpenFaasTrigger, error) {
	openfaastrigger := trigger.Template.OpenFaas

	if _, ok := connectors[openfaastrigger.GatewayURL]; !ok {
		var err error

		username := "admin"
		password := ""

		if openfaastrigger.Username != nil {
			password, err = common.GetSecrets(k8sClient, openfaastrigger.Namespace, openfaastrigger.Username)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to retrieve the username from secret %s and namespace %s", openfaastrigger.Username.Name, openfaastrigger.Namespace)
			}
		}

		if openfaastrigger.Password != nil {
			password, err = common.GetSecrets(k8sClient, openfaastrigger.Namespace, openfaastrigger.Password)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to retrieve the password from secret %s and namespace %s", openfaastrigger.Password.Name, openfaastrigger.Namespace)
			}
		}

		creds := &auth.BasicAuthCredentials{
			User:     username,
			Password: password,
		}

		config := &openfaas.ControllerConfig{
			RebuildInterval: time.Millisecond * 1000,
			GatewayURL:      openfaastrigger.GatewayURL,
		}

		controller := openfaas.NewController(creds, config)

		responseCh := make(chan openfaas.InvokerResponse)

		receiver := ResponseReceiver{
			ResponseCh: responseCh,
		}
		controller.Subscribe(&receiver)

		controller.BeginMapBuilder()

		connectors[openfaastrigger.GatewayURL] = &types.OpenFaasContext{
			Controller: controller,
			ResponseCh: responseCh,
		}
	}

	return &OpenFaasTrigger{
		K8sClient:       k8sClient,
		Sensor:          sensor,
		Trigger:         trigger,
		Logger:          logger,
		OpenFaasContext: connectors[openfaastrigger.GatewayURL],
	}, nil
}

func (t *OpenFaasTrigger) FetchResource() (interface{}, error) {
	return t.Trigger.Template.OpenFaas, nil
}

func (t *OpenFaasTrigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the OpenFaas trigger resource")
	}

	parameters := t.Trigger.Template.OpenFaas.Parameters

	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, t.Trigger.Template.OpenFaas.Parameters, triggers.ExtractEvents(sensor, parameters))
		if err != nil {
			return nil, err
		}

		var ht *v1alpha1.OpenFaasTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the updated OpenFaas trigger resource after applying resource parameters")
		}

		return ht, nil
	}

	return resource, nil
}

// Execute executes the trigger
func (t *OpenFaasTrigger) Execute(resource interface{}) (interface{}, error) {
	obj, ok := resource.(*v1alpha1.OpenFaasTrigger)
	if !ok {
		return nil, errors.New("failed to marshal the OpenFaas trigger resource")
	}

	if obj.Payload == nil {
		return nil, errors.New("payload parameters are not specified")
	}

	payload, err := triggers.ConstructPayload(t.Sensor, obj.Payload)
	if err != nil {
		return nil, err
	}

	t.OpenFaasContext.Controller.Invoke(t.Trigger.Template.OpenFaas.FunctionName, &payload)

	t.Logger.WithField("function-name", t.Trigger.Template.OpenFaas.FunctionName).Infoln("invoked the openfaas function")

	response := <-t.OpenFaasContext.ResponseCh

	t.Logger.WithField("body", string(*response.Body)).Infoln("response body")

	return response, nil
}

// ApplyPolicy applies a policy on trigger execution response if any
func (t *OpenFaasTrigger) ApplyPolicy(resource interface{}) error {
	return nil
}
