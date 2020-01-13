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
package aws_lambda

import (
	"encoding/json"

	commonaws "github.com/argoproj/argo-events/gateways/server/common/aws"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// AWSLambdaTrigger refers to trigger that invokes AWS Lambda functions
type AWSLambdaTrigger struct {
	// K8sClient is Kubernetes client
	K8sClient kubernetes.Interface
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger definition
	Trigger *v1alpha1.Trigger
	// logger to log stuff
	Logger *logrus.Logger
}

// NewAWSLambdaTrigger returns a new AWS Lambda context
func NewAWSLambdaTrigger(k8sClient kubernetes.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger) *AWSLambdaTrigger {
	return &AWSLambdaTrigger{
		K8sClient: k8sClient,
		Sensor:    sensor,
		Trigger:   trigger,
		Logger:    logger,
	}
}

// FetchResource fetches the trigger resource
func (t *AWSLambdaTrigger) FetchResource() (interface{}, error) {
	return t.Trigger.Template.AWSLambda, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *AWSLambdaTrigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the aws lamda trigger resource")
	}
	parameters := t.Trigger.Template.AWSLambda.Parameters
	if parameters != nil && len(parameters) > 0 {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, t.Trigger.Template.AWSLambda.Parameters, triggers.ExtractEvents(sensor, parameters))
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.AWSLambdaTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the updated aws lambda trigger resource after applying resource parameters")
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *AWSLambdaTrigger) Execute(resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.AWSLambdaTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}

	if trigger.Payload == nil {
		return nil, errors.New("payload parameters are not specified")
	}

	payload := make(map[string][]byte)

	events := triggers.ExtractEvents(t.Sensor, trigger.Payload)
	if events == nil {
		return nil, errors.New("payload can't be constructed as there are not events to extract data from")
	}

	for _, parameter := range trigger.Payload {
		value, err := triggers.ResolveParamValue(parameter.Src, events)
		if err != nil {
			return nil, err
		}
		payload[parameter.Dest] = []byte(value)
	}

	payloadBody, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload")
	}

	awsSession, err := commonaws.CreateAWSSession(t.K8sClient, trigger.Namespace, trigger.Region, trigger.AccessKey, trigger.SecretKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a AWS session")
	}

	client := lambda.New(awsSession, &aws.Config{Region: &trigger.Region})

	response, err := client.Invoke(&lambda.InvokeInput{
		FunctionName: &trigger.FunctionName,
		Payload:      payloadBody,
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

// ApplyPolicy applies the policy on the trigger execution response
func (t *AWSLambdaTrigger) ApplyPolicy(resource interface{}) error {
	if t.Trigger.Policy == nil || t.Trigger.Policy.Status == nil || t.Trigger.Policy.Status.Allow == nil {
		return nil
	}

	obj, ok := resource.(*lambda.InvokeOutput)
	if !ok {
		return errors.New("failed to interpret the trigger resource")
	}

	p := policy.NewStatusPolicy(int(*obj.StatusCode), t.Trigger.Policy.Status.Allow)
	return p.ApplyPolicy()
}
