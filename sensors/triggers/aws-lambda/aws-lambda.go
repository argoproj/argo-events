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
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	commonaws "github.com/argoproj/argo-events/eventsources/common/aws"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
)

// AWSLambdaTrigger refers to trigger that invokes AWS Lambda functions
type AWSLambdaTrigger struct {
	// LambdaClient is AWS Lambda client
	LambdaClient *lambda.Lambda
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger definition
	Trigger *v1alpha1.Trigger
	// logger to log stuff
	Logger *zap.SugaredLogger
}

// NewAWSLambdaTrigger returns a new AWS Lambda context
func NewAWSLambdaTrigger(lambdaClients common.StringKeyedMap[*lambda.Lambda], sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) (*AWSLambdaTrigger, error) {
	lambdatrigger := trigger.Template.AWSLambda

	lambdaClient, ok := lambdaClients.Load(trigger.Template.Name)
	if !ok {
		awsSession, err := commonaws.CreateAWSSessionWithCredsInVolume(lambdatrigger.Region, lambdatrigger.RoleARN, lambdatrigger.AccessKey, lambdatrigger.SecretKey, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create a AWS session, %w", err)
		}
		lambdaClient = lambda.New(awsSession, &aws.Config{Region: &lambdatrigger.Region})
		lambdaClients.Store(trigger.Template.Name, lambdaClient)
	}

	return &AWSLambdaTrigger{
		LambdaClient: lambdaClient,
		Sensor:       sensor,
		Trigger:      trigger,
		Logger:       logger.With(logging.LabelTriggerType, apicommon.LambdaTrigger),
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *AWSLambdaTrigger) GetTriggerType() apicommon.TriggerType {
	return apicommon.LambdaTrigger
}

// FetchResource fetches the trigger resource
func (t *AWSLambdaTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.AWSLambda, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *AWSLambdaTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the aws lamda trigger resource, %w", err)
	}
	parameters := t.Trigger.Template.AWSLambda.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, t.Trigger.Template.AWSLambda.Parameters, events)
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.AWSLambdaTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated aws lambda trigger resource after applying resource parameters, %w", err)
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *AWSLambdaTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.AWSLambdaTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the trigger resource")
	}

	if trigger.Payload == nil {
		return nil, fmt.Errorf("payload parameters are not specified")
	}

	payload, err := triggers.ConstructPayload(events, trigger.Payload)
	if err != nil {
		return nil, err
	}

	response, err := t.LambdaClient.Invoke(&lambda.InvokeInput{
		FunctionName:   &trigger.FunctionName,
		Payload:        payload,
		InvocationType: trigger.InvocationType,
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

// ApplyPolicy applies the policy on the trigger execution response
func (t *AWSLambdaTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	if t.Trigger.Policy == nil || t.Trigger.Policy.Status == nil || t.Trigger.Policy.Status.Allow == nil {
		return nil
	}

	obj, ok := resource.(*lambda.InvokeOutput)
	if !ok {
		return fmt.Errorf("failed to interpret the trigger resource")
	}

	p := policy.NewStatusPolicy(int(*obj.StatusCode), t.Trigger.Policy.Status.GetAllow())
	return p.ApplyPolicy(ctx)
}
