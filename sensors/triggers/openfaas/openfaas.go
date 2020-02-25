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
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

const (
	EnvVarOpenFaasGatewayURL = "OPENFAAS_URL"
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
}

// NewOpenFaasTrigger returns a new OpenFaas trigger context
func NewOpenFaasTrigger(k8sClient kubernetes.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger) *OpenFaasTrigger {
	return &OpenFaasTrigger{
		K8sClient: k8sClient,
		Sensor:    sensor,
		Trigger:   trigger,
		Logger:    logger,
	}
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

	if err := os.Setenv(EnvVarOpenFaasGatewayURL, obj.GatewayURL); err != nil {
		return nil, errors.Wrapf(err, "failed to set environment variable OPENFAAS_URL to %s", obj.GatewayURL)
	}

	if obj.Password != nil {
		password, err := common.GetSecrets(t.K8sClient, obj.Namespace, obj.Password)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve the password from secret %s and namespace %s", obj.Password.Name, obj.Namespace)
		}

		cmd := exec.Command("faas", "login", "--password", password)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, errors.Wrap(err, "failed to login using faas client")
		}
	}

	functionURL := fmt.Sprintf("%s/function/%s", obj.GatewayURL, obj.FunctionName)

	parsedURL, err := url.Parse(functionURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse url %s", parsedURL)
	}

	client := http.Client{
		Timeout: 1 * time.Minute,
	}

	request, err := http.NewRequest(http.MethodPost, parsedURL.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create the function request %s", obj.FunctionName)
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrapf(err, "function invocation %s failed", obj.FunctionName)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read the response")
	}

	t.Logger.WithField("body", string(body)).Infoln("response body")

	return body, nil
}

// ApplyPolicy applies a policy on trigger execution response if any
func (t *OpenFaasTrigger) ApplyPolicy(resource interface{}) error {
	return nil
}
