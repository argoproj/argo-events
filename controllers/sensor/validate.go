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

package sensor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/pkg/errors"
)

// ValidateSensor accepts a sensor and performs validation against it
// we return an error so that it can be logged as a message on the sensor status
// the error is ignored by the operation context as subsequent re-queues would produce the same error.
// Exporting this function so that external APIs can use this to validate sensor resource.
func ValidateSensor(s *v1alpha1.Sensor) error {
	if err := validateDependencies(s.Spec.Dependencies); err != nil {
		return err
	}
	err := validateTriggers(s.Spec.Triggers)
	if err != nil {
		return err
	}
	if s.Spec.Template == nil {
		return errors.Errorf("sensor pod template not defined")
	}
	if s.Spec.Subscription == nil {
		return errors.New("at least one subscription must be specified")
	}
	if err := validateSubscription(s.Spec.Subscription); err != nil {
		return errors.Wrap(err, "subscription is invalid")
	}
	if s.Spec.DependencyGroups != nil {
		if s.Spec.Circuit == "" {
			return errors.Errorf("no circuit expression provided to resolve dependency groups")
		}
		expression, err := govaluate.NewEvaluableExpression(s.Spec.Circuit)
		if err != nil {
			return errors.Errorf("circuit expression can't be created for dependency groups. err: %+v", err)
		}

		groups := make(map[string]interface{}, len(s.Spec.DependencyGroups))
		for _, group := range s.Spec.DependencyGroups {
			groups[group.Name] = false
		}
		if _, err = expression.Evaluate(groups); err != nil {
			return errors.Errorf("circuit expression can't be evaluated for dependency groups. err: %+v", err)
		}
	}

	return nil
}

// validateSubscription validates the sensor subscription
func validateSubscription(subscription *v1alpha1.Subscription) error {
	if subscription.HTTP == nil && subscription.NATS == nil {
		return errors.New("either HTTP or NATS subscription must be specified")
	}
	if subscription.NATS != nil {
		if subscription.NATS.ServerURL == "" {
			return errors.New("NATS server url must be specified for the subscription")
		}
		if subscription.NATS.Subject == "" {
			return errors.New("NATS subject must be specified for the subscription")
		}
	}
	return nil
}

// validateTriggers validates triggers
func validateTriggers(triggers []v1alpha1.Trigger) error {
	if len(triggers) < 1 {
		return errors.Errorf("no triggers found")
	}

	for _, trigger := range triggers {
		if err := validateTriggerTemplate(trigger.Template); err != nil {
			return err
		}
		if err := validateTriggerPolicy(&trigger); err != nil {
			return err
		}
		if err := validateTriggerTemplateParameters(&trigger); err != nil {
			return err
		}
	}
	return nil
}

// validateTriggerTemplate validates trigger template
func validateTriggerTemplate(template *v1alpha1.TriggerTemplate) error {
	if template == nil {
		return errors.Errorf("trigger template can't be nil")
	}
	if template.Name == "" {
		return errors.Errorf("trigger must define a name")
	}
	if template.Switch != nil && template.Switch.All != nil && template.Switch.Any != nil {
		return errors.Errorf("trigger condition can't have both any and all condition")
	}
	if template.K8s != nil {
		if err := validateK8sTrigger(template.K8s); err != nil {
			return errors.Wrapf(err, "trigger for template %s is invalid", template.Name)
		}
	}
	if template.ArgoWorkflow != nil {
		if err := validateArgoWorkflowTrigger(template.ArgoWorkflow); err != nil {
			return errors.Wrapf(err, "template %s is invalid", template.Name)
		}
	}
	if template.HTTP != nil {
		if err := validateHTTPTrigger(template.HTTP); err != nil {
			return errors.Wrapf(err, "template %s is invalid", template.Name)
		}
	}
	if template.AWSLambda != nil {
		if err := validateAWSLambdaTrigger(template.AWSLambda); err != nil {
			return errors.Wrapf(err, "template %s is invalid", template.Name)
		}
	}
	if template.Kafka != nil {
		if err := validateKafkaTrigger(template.Kafka); err != nil {
			return errors.Wrapf(err, "template %s is invalid", template.Name)
		}
	}
	if template.NATS != nil {
		if err := validateNATSTrigger(template.NATS); err != nil {
			return errors.Wrapf(err, "template %s is invalid", template.Name)
		}
	}
	if template.Slack != nil {
		if err := validateSlackTrigger(template.Slack); err != nil {
			return errors.Wrapf(err, "template %s is invalid", template.Name)
		}
	}
	if template.OpenWhisk != nil {
		if err := validateOpenWhiskTrigger(template.OpenWhisk); err != nil {
			return errors.Wrapf(err, "template %s is invalid", template.Name)
		}
	}
	if template.CustomTrigger != nil {
		if err := validateCustomTrigger(template.CustomTrigger); err != nil {
			return errors.Wrapf(err, "template %s is invalid", template.Name)
		}
	}
	return nil
}

// validateK8sTrigger validates a kubernetes trigger
func validateK8sTrigger(trigger *v1alpha1.StandardK8sTrigger) error {
	if trigger == nil {
		return errors.New("k8s trigger for can't be nil")
	}
	if trigger.Source == nil {
		return errors.New("k8s trigger for does not contain an absolute action")
	}
	if trigger.GroupVersionResource == nil {
		return errors.New("must provide group, version and resource for the resource")
	}
	switch trigger.Operation {
	case v1alpha1.Create, v1alpha1.Update:
	default:
		return errors.Errorf("unknown operation type %s", string(trigger.Operation))
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return errors.Errorf("resource parameter index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateArgoWorkflowTrigger validates an Argo workflow trigger
func validateArgoWorkflowTrigger(trigger *v1alpha1.ArgoWorkflowTrigger) error {
	if trigger == nil {
		return errors.New("k8s trigger for can't be nil")
	}
	if trigger.Source == nil {
		return errors.New("k8s trigger for does not contain an absolute action")
	}
	if trigger.GroupVersionResource == nil {
		return errors.New("must provide group, version and resource for the resource")
	}
	switch trigger.Operation {
	case v1alpha1.Submit, v1alpha1.Suspend, v1alpha1.Retry, v1alpha1.Resume, v1alpha1.Resubmit:
	default:
		return errors.Errorf("unknown operation type %s", string(trigger.Operation))
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return errors.Errorf("resource parameter index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateHTTPTrigger validates the HTTP trigger
func validateHTTPTrigger(trigger *v1alpha1.HTTPTrigger) error {
	if trigger == nil {
		return errors.New("openfaas trigger for can't be nil")
	}
	if trigger.URL == "" {
		return errors.New("server URL is not specified")
	}
	if trigger.Method != "" {
		switch trigger.Method {
		case http.MethodGet, http.MethodDelete, http.MethodPatch, http.MethodPost, http.MethodPut:
		default:
			return errors.New("only GET, DELETE, PATCH, POST and PUT methods are supported")
		}
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return errors.Errorf("resource parameter index: %d. err: %+v", i, err)
			}
		}
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return errors.Errorf("payload index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateOpenWhiskTrigger validates the OpenWhisk trigger
func validateOpenWhiskTrigger(trigger *v1alpha1.OpenWhiskTrigger) error {
	if trigger == nil {
		return errors.New("openwhisk trigger for can't be nil")
	}
	if trigger.ActionName == "" {
		return errors.New("action name is not specified")
	}
	if trigger.Host == "" {
		return errors.New("host URL is not specified")
	}
	if trigger.AuthToken != nil {
		if trigger.AuthToken.Name == "" || trigger.AuthToken.Key == "" {
			return errors.New("auth token key and name must be specified")
		}
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return errors.Errorf("resource parameter index: %d. err: %+v", i, err)
			}
		}
	}
	if trigger.Payload == nil {
		return errors.New("payload parameters are not specified")
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return errors.Errorf("resource parameter index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateAWSLambdaTrigger validates the AWS Lambda trigger
func validateAWSLambdaTrigger(trigger *v1alpha1.AWSLambdaTrigger) error {
	if trigger == nil {
		return errors.New("openfaas trigger for can't be nil")
	}
	if trigger.FunctionName == "" {
		return errors.New("function name is not specified")
	}
	if trigger.Region == "" {
		return errors.New("region in not specified")
	}
	if trigger.AccessKey == nil || trigger.SecretKey == nil {
		return errors.New("either accesskey or secretkey secret selector is not specified")
	}
	if trigger.Payload == nil {
		return errors.New("payload parameters are not specified")
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return errors.Errorf("resource parameter index: %d. err: %+v", i, err)
			}
		}
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return errors.Errorf("payload index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateKafkaTrigger validates the kafka trigger.
func validateKafkaTrigger(trigger *v1alpha1.KafkaTrigger) error {
	if trigger == nil {
		return errors.New("trigger can't be nil")
	}
	if trigger.URL == "" {
		return errors.New("broker url must not be empty")
	}
	if trigger.Payload == nil {
		return errors.New("payload must not be empty")
	}
	if trigger.Topic == "" {
		return errors.New("topic must not be empty")
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return errors.Errorf("payload index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateNATSTrigger validates the NATS trigger.
func validateNATSTrigger(trigger *v1alpha1.NATSTrigger) error {
	if trigger == nil {
		return errors.New("trigger can't be nil")
	}
	if trigger.URL == "" {
		return errors.New("nats server url can't be empty")
	}
	if trigger.Subject == "" {
		return errors.New("nats subject can't be empty")
	}
	if trigger.Payload == nil {
		return errors.New("payload can't be nil")
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return errors.Errorf("payload index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateSlackTrigger validates the Slack trigger.
func validateSlackTrigger(trigger *v1alpha1.SlackTrigger) error {
	if trigger == nil {
		return errors.New("trigger can't be nil")
	}
	if trigger.SlackToken == nil {
		return errors.New("slack token can't be empty")
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return errors.Errorf("resource parameter index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateCustomTrigger validates the custom trigger.
func validateCustomTrigger(trigger *v1alpha1.CustomTrigger) error {
	if trigger == nil {
		return errors.New("custom trigger for can't be nil")
	}
	if trigger.ServerURL == "" {
		return errors.New("custom trigger gRPC server url is not defined")
	}
	if trigger.Spec == nil {
		return errors.New("trigger body can't be empty")
	}
	if trigger.Secure {
		if trigger.CertFilePath == "" {
			return errors.New("cert file path can't be nil when the trigger server connection is secure")
		}
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return errors.Errorf("resource parameter index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateTriggerParameters validates resource and template parameters if any
func validateTriggerTemplateParameters(trigger *v1alpha1.Trigger) error {
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return errors.Errorf("template parameter index: %d. err: %+v", i, err)
			}
		}
	}
	return nil
}

// validateTriggerParameter validates a trigger parameter
func validateTriggerParameter(parameter *v1alpha1.TriggerParameter) error {
	if parameter.Src == nil {
		return errors.Errorf("parameter source can't be empty")
	}
	if parameter.Src.DependencyName == "" {
		return errors.Errorf("parameter dependency name can't be empty")
	}
	if parameter.Dest == "" {
		return errors.Errorf("parameter destination can't be empty")
	}

	switch op := parameter.Operation; op {
	case v1alpha1.TriggerParameterOpAppend:
	case v1alpha1.TriggerParameterOpOverwrite:
	case v1alpha1.TriggerParameterOpPrepend:
	case v1alpha1.TriggerParameterOpNone:
	default:
		return errors.Errorf("parameter operation %+v is invalid", op)
	}

	return nil
}

// perform a check to see that each event dependency is in correct format and has valid filters set if any
func validateDependencies(eventDependencies []v1alpha1.EventDependency) error {
	if len(eventDependencies) < 1 {
		return errors.New("no event dependencies found")
	}
	for _, dep := range eventDependencies {
		if dep.Name == "" {
			return errors.New("event dependency must define a name")
		}

		if dep.GatewayName == "" {
			return errors.New("event dependency must define the gateway name")
		}

		if dep.EventName == "" {
			return errors.New("event dependency must define the event name")
		}

		if err := validateEventFilter(dep.Filters); err != nil {
			return err
		}
	}
	return nil
}

// validateEventFilter for a sensor
func validateEventFilter(filter *v1alpha1.EventDependencyFilter) error {
	if filter == nil {
		return nil
	}
	if filter.Time != nil {
		if err := validateEventTimeFilter(filter.Time); err != nil {
			return err
		}
	}
	return nil
}

// validateEventTimeFilter validates time filter
func validateEventTimeFilter(tFilter *v1alpha1.TimeFilter) error {
	currentT := time.Now().UTC()
	currentT = time.Date(currentT.Year(), currentT.Month(), currentT.Day(), 0, 0, 0, 0, time.UTC)
	currentTStr := currentT.Format(common.StandardYYYYMMDDFormat)
	if tFilter.Start != "" && tFilter.Stop != "" {
		startTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTStr, tFilter.Start))
		if err != nil {
			return err
		}
		stopTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTStr, tFilter.Stop))
		if err != nil {
			return err
		}
		if stopTime.Before(startTime) || startTime.Equal(stopTime) {
			return errors.Errorf("invalid event time filter: stop '%s' is before or equal to start '%s", tFilter.Stop, tFilter.Start)
		}
	}
	if tFilter.Stop != "" {
		stopTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTStr, tFilter.Stop))
		if err != nil {
			return err
		}
		stopTime = stopTime.UTC()
		if stopTime.Before(currentT.UTC()) {
			return errors.Errorf("invalid event time filter: stop '%s' is before the current time '%s'", tFilter.Stop, currentT)
		}
	}
	return nil
}

// validateTriggerPolicy validates a trigger policy
func validateTriggerPolicy(trigger *v1alpha1.Trigger) error {
	if trigger.Policy == nil {
		return nil
	}
	if trigger.Template.K8s != nil {
		return validateK8sTriggerPolicy(trigger.Policy.K8s)
	}
	if trigger.Template.ArgoWorkflow != nil {
		return validateK8sTriggerPolicy(trigger.Policy.K8s)
	}
	if trigger.Template.HTTP != nil {
		return validateStatusPolicy(trigger.Policy.Status)
	}
	if trigger.Template.AWSLambda != nil {
		return validateStatusPolicy(trigger.Policy.Status)
	}
	return nil
}

// validateK8sTriggerPolicy validates a k8s trigger policy
func validateK8sTriggerPolicy(policy *v1alpha1.K8sResourcePolicy) error {
	if policy == nil {
		return nil
	}
	if policy.Labels == nil {
		return errors.New("resource labels are not specified")
	}
	if &policy.Backoff == nil {
		return errors.New("backoff is not specified")
	}
	return nil
}

// validateStatusPolicy validates a http trigger policy
func validateStatusPolicy(policy *v1alpha1.StatusPolicy) error {
	if policy == nil {
		return nil
	}
	if policy.Allow == nil {
		return errors.New("list of allowed response status is not specified")
	}
	return nil
}
