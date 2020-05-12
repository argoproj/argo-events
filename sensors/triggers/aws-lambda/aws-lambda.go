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
	// LambdaClient is AWS Lambda client
	LambdaClient *lambda.Lambda
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
func NewAWSLambdaTrigger(lambdaClients map[string]*lambda.Lambda, k8sClient kubernetes.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger) (*AWSLambdaTrigger, error) {
	lambdatrigger := trigger.Template.AWSLambda

	lambdaClient, ok := lambdaClients[trigger.Template.Name]
	if !ok {
		awsSession, err := commonaws.CreateAWSSession(k8sClient, lambdatrigger.Namespace, lambdatrigger.Region, "", lambdatrigger.AccessKey, lambdatrigger.SecretKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create a AWS session")
		}
		lambdaClient = lambda.New(awsSession, &aws.Config{Region: &lambdatrigger.Region})
		lambdaClients[trigger.Template.Name] = lambdaClient
	}

	if lambdatrigger.Namespace == "" {
		lambdatrigger.Namespace = sensor.Namespace
	}

	return &AWSLambdaTrigger{
		LambdaClient: lambdaClient,
		K8sClient:    k8sClient,
		Sensor:       sensor,
		Trigger:      trigger,
		Logger:       logger,
	}, nil
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
	if parameters != nil {
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

	payload, err := triggers.ConstructPayload(t.Sensor, trigger.Payload)
	if err != nil {
		return nil, err
	}

	response, err := t.LambdaClient.Invoke(&lambda.InvokeInput{
		FunctionName: &trigger.FunctionName,
		Payload:      payload,
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
