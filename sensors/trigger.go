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
	"context"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	argoworkflow "github.com/argoproj/argo-events/sensors/triggers/argo-workflow"
	awslambda "github.com/argoproj/argo-events/sensors/triggers/aws-lambda"
	customtrigger "github.com/argoproj/argo-events/sensors/triggers/custom-trigger"
	"github.com/argoproj/argo-events/sensors/triggers/http"
	"github.com/argoproj/argo-events/sensors/triggers/kafka"
	"github.com/argoproj/argo-events/sensors/triggers/nats"
	"github.com/argoproj/argo-events/sensors/triggers/slack"
	standardk8s "github.com/argoproj/argo-events/sensors/triggers/standard-k8s"
	"go.uber.org/zap"
)

// Trigger interface
type Trigger interface {
	// FetchResource fetches the trigger resource from external source
	FetchResource() (interface{}, error)
	// ApplyResourceParameters applies parameters to the trigger resource
	ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error)
	// Execute executes the trigger
	Execute(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error)
	// ApplyPolicy applies the policy on the trigger
	ApplyPolicy(resource interface{}) error
}

// GetTrigger returns a trigger
func (sensorCtx *SensorContext) GetTrigger(ctx context.Context, trigger *v1alpha1.Trigger) Trigger {
	log := logging.FromContext(ctx).Desugar()
	if trigger.Template.K8s != nil {
		return standardk8s.NewStandardK8sTrigger(sensorCtx.KubeClient, sensorCtx.DynamicClient, sensorCtx.Sensor, trigger, log)
	}

	if trigger.Template.ArgoWorkflow != nil {
		return argoworkflow.NewArgoWorkflowTrigger(sensorCtx.KubeClient, sensorCtx.DynamicClient, sensorCtx.Sensor, trigger, log)
	}

	if trigger.Template.HTTP != nil {
		result, err := http.NewHTTPTrigger(sensorCtx.httpClients, sensorCtx.Sensor, trigger, log)
		if err != nil {
			log.Error("failed to invoke the trigger", zap.Any("trigger", trigger.Template.Name), zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.AWSLambda != nil {
		result, err := awslambda.NewAWSLambdaTrigger(sensorCtx.awsLambdaClients, sensorCtx.Sensor, trigger, log)
		if err != nil {
			log.Error("failed to invoke the trigger", zap.Any("trigger", trigger.Template.Name), zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.Kafka != nil {
		result, err := kafka.NewKafkaTrigger(sensorCtx.Sensor, trigger, sensorCtx.kafkaProducers, log)
		if err != nil {
			log.Error("failed to invoke the trigger", zap.Any("trigger", trigger.Template.Name), zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.NATS != nil {
		result, err := nats.NewNATSTrigger(sensorCtx.Sensor, trigger, sensorCtx.natsConnections, log)
		if err != nil {
			log.Error("failed to invoke the trigger", zap.Any("trigger", trigger.Template.Name), zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.Slack != nil {
		result, err := slack.NewSlackTrigger(sensorCtx.Sensor, trigger, log, sensorCtx.slackHTTPClient)
		if err != nil {
			log.Error("failed to invoke the trigger", zap.Any("trigger", trigger.Template.Name), zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.CustomTrigger != nil {
		result, err := customtrigger.NewCustomTrigger(sensorCtx.Sensor, trigger, log, sensorCtx.customTriggerClients)
		if err != nil {
			log.Error("failed to invoke the trigger", zap.Any("trigger", trigger.Template.Name), zap.Error(err))
			return nil
		}
		return result
	}
	return nil
}
