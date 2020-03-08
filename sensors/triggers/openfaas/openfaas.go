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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
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
	// http client to invoke function.
	httpClient *http.Client
}

// NewOpenFaasTrigger returns a new OpenFaas trigger context
func NewOpenFaasTrigger(k8sClient kubernetes.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger, httpClient *http.Client) (*OpenFaasTrigger, error) {
	return &OpenFaasTrigger{
		K8sClient:  k8sClient,
		Sensor:     sensor,
		Trigger:    trigger,
		Logger:     logger,
		httpClient: httpClient,
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

	username := "admin"
	password := ""

	openfaastrigger := t.Trigger.Template.OpenFaas

	if openfaastrigger.Username != nil {
		password, err = common.GetSecrets(t.K8sClient, openfaastrigger.Namespace, openfaastrigger.Username)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve the username from secret %s and namespace %s", openfaastrigger.Username.Name, openfaastrigger.Namespace)
		}
	}

	if openfaastrigger.Password != nil {
		password, err = common.GetSecrets(t.K8sClient, openfaastrigger.Namespace, openfaastrigger.Password)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve the password from secret %s and namespace %s", openfaastrigger.Password.Name, openfaastrigger.Namespace)
		}
	}

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/function/%s", openfaastrigger.GatewayURL, openfaastrigger.FunctionName), bytes.NewReader(payload))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to construct request for function %s", openfaastrigger.FunctionName)
	}
	request.SetBasicAuth(username, password)

	t.Logger.WithField("function", openfaastrigger.FunctionName).Infoln("invoking the function...")

	return t.httpClient.Do(request)
}

// ApplyPolicy applies a policy on trigger execution response if any
func (t *OpenFaasTrigger) ApplyPolicy(resource interface{}) error {
	if t.Trigger.Policy == nil || t.Trigger.Policy.Status == nil || t.Trigger.Policy.Status.Allow == nil {
		return nil
	}
	response, ok := resource.(*http.Response)
	if !ok {
		return errors.New("failed to interpret the trigger execution response")
	}

	p := policy.NewStatusPolicy(response.StatusCode, t.Trigger.Policy.Status.Allow)

	return p.ApplyPolicy()
}
