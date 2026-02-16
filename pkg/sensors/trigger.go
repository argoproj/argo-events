/*
Copyright 2018 The Argoproj Authors.

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

	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	openwhisk "github.com/argoproj/argo-events/pkg/sensors/triggers/apache-openwhisk"
	argoworkflow "github.com/argoproj/argo-events/pkg/sensors/triggers/argo-workflow"
	awslambda "github.com/argoproj/argo-events/pkg/sensors/triggers/aws-lambda"
	eventhubs "github.com/argoproj/argo-events/pkg/sensors/triggers/azure-event-hubs"
	servicebus "github.com/argoproj/argo-events/pkg/sensors/triggers/azure-service-bus"
	customtrigger "github.com/argoproj/argo-events/pkg/sensors/triggers/custom-trigger"
	"github.com/argoproj/argo-events/pkg/sensors/triggers/email"
	gcpcloudfunctions "github.com/argoproj/argo-events/pkg/sensors/triggers/gcp-cloud-functions"
	"github.com/argoproj/argo-events/pkg/sensors/triggers/http"
	"github.com/argoproj/argo-events/pkg/sensors/triggers/kafka"
	logtrigger "github.com/argoproj/argo-events/pkg/sensors/triggers/log"
	"github.com/argoproj/argo-events/pkg/sensors/triggers/nats"
	"github.com/argoproj/argo-events/pkg/sensors/triggers/pulsar"
	"github.com/argoproj/argo-events/pkg/sensors/triggers/slack"
	standardk8s "github.com/argoproj/argo-events/pkg/sensors/triggers/standard-k8s"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

// Trigger interface
type Trigger interface {
	GetTriggerType() v1alpha1.TriggerType
	// FetchResource fetches the trigger resource from external source
	FetchResource(context.Context) (interface{}, error)
	// ApplyResourceParameters applies parameters to the trigger resource
	ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error)
	// Execute executes the trigger
	Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error)
	// ApplyPolicy applies the policy on the trigger
	ApplyPolicy(ctx context.Context, resource interface{}) error
}

// GetTrigger returns a trigger
func (sensorCtx *SensorContext) GetTrigger(ctx context.Context, trigger *v1alpha1.Trigger) Trigger {
	log := logging.FromContext(ctx).With(logging.LabelTriggerName, trigger.Template.Name)
	if trigger.Template.K8s != nil {
		return standardk8s.NewStandardK8sTrigger(sensorCtx.kubeClient, sensorCtx.dynamicClient, sensorCtx.sensor, trigger, log)
	}

	if trigger.Template.ArgoWorkflow != nil {
		return argoworkflow.NewArgoWorkflowTrigger(sensorCtx.kubeClient, sensorCtx.dynamicClient, sensorCtx.sensor, trigger, log)
	}

	if trigger.Template.HTTP != nil {
		result, err := http.NewHTTPTrigger(sensorCtx.httpClients, sensorCtx.sensor, trigger, log)
		if err != nil {
			log.Errorw("failed to new an HTTP trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.AWSLambda != nil {
		result, err := awslambda.NewAWSLambdaTrigger(sensorCtx.awsLambdaClients, sensorCtx.sensor, trigger, log)
		if err != nil {
			log.Errorw("failed to new a Lambda trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.AzureEventHubs != nil {
		result, err := eventhubs.NewAzureEventHubsTrigger(sensorCtx.sensor, trigger, sensorCtx.azureEventHubsClients, log)
		if err != nil {
			log.Errorw("failed to new an Azure Event Hubs trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.AzureServiceBus != nil {
		result, err := servicebus.NewAzureServiceBusTrigger(sensorCtx.sensor, trigger, sensorCtx.azureServiceBusClients, log)
		if err != nil {
			log.Errorw("failed to new an Azure Service Bus trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.Kafka != nil {
		result, err := kafka.NewKafkaTrigger(sensorCtx.sensor, trigger, sensorCtx.kafkaProducers, log)
		if err != nil {
			log.Errorw("failed to new a Kafka trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.Pulsar != nil {
		result, err := pulsar.NewPulsarTrigger(sensorCtx.sensor, trigger, sensorCtx.pulsarProducers, log)
		if err != nil {
			log.Errorw("failed to new a Pulsar trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.NATS != nil {
		result, err := nats.NewNATSTrigger(sensorCtx.sensor, trigger, sensorCtx.natsConnections, log)
		if err != nil {
			log.Errorw("failed to new a NATS trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.Slack != nil {
		result, err := slack.NewSlackTrigger(sensorCtx.sensor, trigger, log, sensorCtx.slackHTTPClient)
		if err != nil {
			log.Errorw("failed to new a Slack trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.OpenWhisk != nil {
		result, err := openwhisk.NewTriggerImpl(sensorCtx.sensor, trigger, sensorCtx.openwhiskClients, log)
		if err != nil {
			log.Errorw("failed to new an OpenWhisk trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.CustomTrigger != nil {
		result, err := customtrigger.NewCustomTrigger(sensorCtx.sensor, trigger, log, sensorCtx.customTriggerClients)
		if err != nil {
			log.Errorw("failed to new a Custom trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.Log != nil {
		result, err := logtrigger.NewLogTrigger(sensorCtx.sensor, trigger, log)
		if err != nil {
			log.Errorw("failed to new a Log trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.Email != nil {
		result, err := email.NewEmailTrigger(sensorCtx.sensor, trigger, log)
		if err != nil {
			log.Errorw("failed to new a Email trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.GCPCloudFunctions != nil {
		result, err := gcpcloudfunctions.NewGCPCloudFunctionsTrigger(sensorCtx.gcpCloudFunctionsClients, sensorCtx.sensor, trigger, log)
		if err != nil {
			log.Errorw("failed to new a GCP Cloud Functions trigger", zap.Error(err))
			return nil
		}
		return result
	}
	return nil
}
