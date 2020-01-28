/*
Copyright 2018 BlackRock, Inc.

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
package sensors

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	argoworkflow "github.com/argoproj/argo-events/sensors/triggers/argo-workflow"
	aws_lambda "github.com/argoproj/argo-events/sensors/triggers/aws-lambda"
	"github.com/argoproj/argo-events/sensors/triggers/http"
	"github.com/argoproj/argo-events/sensors/triggers/openfaas"
	standardk8s "github.com/argoproj/argo-events/sensors/triggers/standard-k8s"
)

// Trigger interface
type Trigger interface {
	// FetchResource fetches the trigger resource from external source
	FetchResource() (interface{}, error)
	// ApplyResourceParameters applies parameters to the trigger resource
	ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error)
	// Execute executes the trigger
	Execute(resource interface{}) (interface{}, error)
	// ApplyPolicy applies the policy on the trigger
	ApplyPolicy(resource interface{}) error
}

// GetTrigger returns a trigger
func (sensorCtx *SensorContext) GetTrigger(trigger *v1alpha1.Trigger) Trigger {
	if trigger.Template.K8s != nil {
		return standardk8s.NewStandardK8sTrigger(sensorCtx.KubeClient, sensorCtx.DynamicClient, sensorCtx.Sensor, trigger, sensorCtx.Logger)
	}
	if trigger.Template.ArgoWorkflow != nil {
		return argoworkflow.NewArgoWorkflowTrigger(sensorCtx.KubeClient, sensorCtx.DynamicClient, sensorCtx.Sensor, trigger, sensorCtx.Logger)
	}
	if trigger.Template.OpenFaas != nil {
		return openfaas.NewOpenFaasTrigger(sensorCtx.KubeClient, sensorCtx.Sensor, trigger, sensorCtx.Logger)
	}
	if trigger.Template.HTTP != nil {
		return http.NewHTTPTrigger(sensorCtx.Sensor, trigger, sensorCtx.Logger)
	}
	if trigger.Template.AWSLambda != nil {
		return aws_lambda.NewAWSLambdaTrigger(sensorCtx.KubeClient, sensorCtx.Sensor, trigger, sensorCtx.Logger)
	}
	return nil
}
