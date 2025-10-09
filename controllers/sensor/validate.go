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

	cronlib "github.com/robfig/cron/v3"

	"github.com/argoproj/argo-events/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// ValidateSensor accepts a sensor and performs validation against it
// we return an error so that it can be logged as a message on the sensor status
// the error is ignored by the operation context as subsequent re-queues would produce the same error.
// Exporting this function so that external APIs can use this to validate sensor resource.
func ValidateSensor(s *v1alpha1.Sensor, b *eventbusv1alpha1.EventBus) error {
	if s == nil {
		s.Status.MarkDependenciesNotProvided("InvalidSensor", "nil sensor")
		return fmt.Errorf("nil sensor")
	}
	if b == nil {
		s.Status.MarkDependenciesNotProvided("InvalidEventBus", "nil eventbus")
		return fmt.Errorf("nil eventbus")
	}
	if err := validateDependencies(s.Spec.Dependencies, b); err != nil {
		s.Status.MarkDependenciesNotProvided("InvalidDependencies", err.Error())
		return err
	}
	s.Status.MarkDependenciesProvided()
	err := validateTriggers(s.Spec.Triggers)
	if err != nil {
		s.Status.MarkTriggersNotProvided("InvalidTriggers", err.Error())
		return err
	}
	s.Status.MarkTriggersProvided()
	return nil
}

// validateTriggers validates triggers
func validateTriggers(triggers []v1alpha1.Trigger) error {
	if len(triggers) < 1 {
		return fmt.Errorf("no triggers found")
	}

	trigNames := make(map[string]bool)

	for _, trigger := range triggers {
		if err := validateTriggerTemplate(trigger.Template); err != nil {
			return err
		}
		if _, ok := trigNames[trigger.Template.Name]; ok {
			return fmt.Errorf("duplicate trigger name: %s", trigger.Template.Name)
		}
		trigNames[trigger.Template.Name] = true
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
		return fmt.Errorf("trigger template can't be nil")
	}
	if template.Name == "" {
		return fmt.Errorf("trigger must define a name")
	}
	if len(template.ConditionsReset) > 0 {
		for _, c := range template.ConditionsReset {
			if c.ByTime == nil {
				return fmt.Errorf("invalid conditionsReset")
			}
			parser := cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow)
			if _, err := parser.Parse(c.ByTime.Cron); err != nil {
				return fmt.Errorf("invalid cron expression %q", c.ByTime.Cron)
			}
			if _, err := time.LoadLocation(c.ByTime.Timezone); err != nil {
				return fmt.Errorf("invalid timezone %q", c.ByTime.Timezone)
			}
		}
	}
	if template.K8s != nil {
		if err := validateK8STrigger(template.K8s); err != nil {
			return fmt.Errorf("trigger for template %s is invalid, %w", template.Name, err)
		}
	}
	if template.ArgoWorkflow != nil {
		if err := validateArgoWorkflowTrigger(template.ArgoWorkflow); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	if template.HTTP != nil {
		if err := validateHTTPTrigger(template.HTTP); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	if template.AWSLambda != nil {
		if err := validateAWSLambdaTrigger(template.AWSLambda); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	if template.Kafka != nil {
		if err := validateKafkaTrigger(template.Kafka); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	if template.NATS != nil {
		if err := validateNATSTrigger(template.NATS); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	if template.Slack != nil {
		if err := validateSlackTrigger(template.Slack); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	if template.OpenWhisk != nil {
		if err := validateOpenWhiskTrigger(template.OpenWhisk); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	if template.CustomTrigger != nil {
		if err := validateCustomTrigger(template.CustomTrigger); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	return nil
}

// validateK8STrigger validates a kubernetes trigger
func validateK8STrigger(trigger *v1alpha1.StandardK8STrigger) error {
	if trigger == nil {
		return fmt.Errorf("k8s trigger can't be nil")
	}
	if trigger.Source == nil {
		return fmt.Errorf("k8s trigger does not contain an absolute action")
	}

	switch trigger.Operation {
	case "", v1alpha1.Create, v1alpha1.Patch, v1alpha1.Update, v1alpha1.Delete:

	default:
		return fmt.Errorf("unknown operation type %s", string(trigger.Operation))
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateArgoWorkflowTrigger validates an Argo workflow trigger
func validateArgoWorkflowTrigger(trigger *v1alpha1.ArgoWorkflowTrigger) error {
	if trigger == nil {
		return fmt.Errorf("argoWorkflow trigger can't be nil")
	}
	if trigger.Source == nil {
		return fmt.Errorf("argoWorkflow trigger does not contain an absolute action")
	}

	switch trigger.Operation {
	case v1alpha1.Submit, v1alpha1.SubmitFrom, v1alpha1.Suspend, v1alpha1.Retry, v1alpha1.Resume, v1alpha1.Resubmit, v1alpha1.Terminate, v1alpha1.Stop:
	default:
		return fmt.Errorf("unknown operation type %s", string(trigger.Operation))
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateHTTPTrigger validates the HTTP trigger
func validateHTTPTrigger(trigger *v1alpha1.HTTPTrigger) error {
	if trigger == nil {
		return fmt.Errorf("HTTP trigger for can't be nil")
	}
	if trigger.URL == "" {
		return fmt.Errorf("server URL is not specified")
	}
	if trigger.Method != "" {
		switch trigger.Method {
		case http.MethodGet, http.MethodDelete, http.MethodPatch, http.MethodPost, http.MethodPut:
		default:
			return fmt.Errorf("only GET, DELETE, PATCH, POST and PUT methods are supported")
		}
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return fmt.Errorf("payload index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateOpenWhiskTrigger validates the OpenWhisk trigger
func validateOpenWhiskTrigger(trigger *v1alpha1.OpenWhiskTrigger) error {
	if trigger == nil {
		return fmt.Errorf("openwhisk trigger for can't be nil")
	}
	if trigger.ActionName == "" {
		return fmt.Errorf("action name is not specified")
	}
	if trigger.Host == "" {
		return fmt.Errorf("host URL is not specified")
	}
	if trigger.AuthToken != nil {
		if trigger.AuthToken.Name == "" || trigger.AuthToken.Key == "" {
			return fmt.Errorf("auth token key and name must be specified")
		}
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	if trigger.Payload == nil {
		return fmt.Errorf("payload parameters are not specified")
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateAWSLambdaTrigger validates the AWS Lambda trigger
func validateAWSLambdaTrigger(trigger *v1alpha1.AWSLambdaTrigger) error {
	if trigger == nil {
		return fmt.Errorf("openfaas trigger for can't be nil")
	}
	if trigger.FunctionName == "" {
		return fmt.Errorf("function name is not specified")
	}
	if trigger.Region == "" {
		return fmt.Errorf("region in not specified")
	}
	if trigger.Payload == nil {
		return fmt.Errorf("payload parameters are not specified")
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return fmt.Errorf("payload index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateKafkaTrigger validates the kafka trigger.
func validateKafkaTrigger(trigger *v1alpha1.KafkaTrigger) error {
	if trigger == nil {
		return fmt.Errorf("trigger can't be nil")
	}
	if trigger.URL == "" {
		return fmt.Errorf("broker url must not be empty")
	}
	if trigger.Payload == nil {
		return fmt.Errorf("payload must not be empty")
	}
	if trigger.Topic == "" {
		return fmt.Errorf("topic must not be empty")
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return fmt.Errorf("payload index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateNATSTrigger validates the NATS trigger.
func validateNATSTrigger(trigger *v1alpha1.NATSTrigger) error {
	if trigger == nil {
		return fmt.Errorf("trigger can't be nil")
	}
	if trigger.URL == "" {
		return fmt.Errorf("nats server url can't be empty")
	}
	if trigger.Subject == "" {
		return fmt.Errorf("nats subject can't be empty")
	}
	if trigger.Payload == nil {
		return fmt.Errorf("payload can't be nil")
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return fmt.Errorf("payload index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateSlackTrigger validates the Slack trigger.
func validateSlackTrigger(trigger *v1alpha1.SlackTrigger) error {
	if trigger == nil {
		return fmt.Errorf("trigger can't be nil")
	}
	if trigger.SlackToken == nil {
		return fmt.Errorf("slack token can't be empty")
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateCustomTrigger validates the custom trigger.
func validateCustomTrigger(trigger *v1alpha1.CustomTrigger) error {
	if trigger == nil {
		return fmt.Errorf("custom trigger for can't be nil")
	}
	if trigger.ServerURL == "" {
		return fmt.Errorf("custom trigger gRPC server url is not defined")
	}
	if trigger.Spec == nil {
		return fmt.Errorf("trigger body can't be empty")
	}
	if trigger.Secure {
		if trigger.CertSecret == nil {
			return fmt.Errorf("certSecret can't be nil when the trigger server connection is secure")
		}
	}
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateTriggerTemplateParameters validates resource and template parameters if any
func validateTriggerTemplateParameters(trigger *v1alpha1.Trigger) error {
	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("template parameter index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}

// validateTriggerParameter validates a trigger parameter
func validateTriggerParameter(parameter *v1alpha1.TriggerParameter) error {
	if parameter.Src == nil {
		return fmt.Errorf("parameter source can't be empty")
	}
	if parameter.Src.DependencyName == "" {
		return fmt.Errorf("parameter dependency name can't be empty")
	}
	if parameter.Dest == "" {
		return fmt.Errorf("parameter destination can't be empty")
	}

	switch op := parameter.Operation; op {
	case v1alpha1.TriggerParameterOpAppend:
	case v1alpha1.TriggerParameterOpOverwrite:
	case v1alpha1.TriggerParameterOpPrepend:
	case v1alpha1.TriggerParameterOpNone:
	default:
		return fmt.Errorf("parameter operation %+v is invalid", op)
	}

	return nil
}

// perform a check to see that each event dependency is in correct format and has valid filters set if any
func validateDependencies(eventDependencies []v1alpha1.EventDependency, b *eventbusv1alpha1.EventBus) error {
	if len(eventDependencies) < 1 {
		return fmt.Errorf("no event dependencies found")
	}

	comboKeys := make(map[string]bool)
	for _, dep := range eventDependencies {
		if dep.Name == "" {
			return fmt.Errorf("event dependency must define a name")
		}
		if dep.EventSourceName == "" {
			return fmt.Errorf("event dependency must define the EventSourceName")
		}

		if dep.EventName == "" {
			return fmt.Errorf("event dependency must define the EventName")
		}
		if b.Spec.NATS != nil {
			// For STAN, EventSourceName + EventName can not be referenced more than once in one Sensor object.
			comboKey := fmt.Sprintf("%s-$$$-%s", dep.EventSourceName, dep.EventName)
			if _, existing := comboKeys[comboKey]; existing {
				return fmt.Errorf("event '%s' from EventSource '%s' is referenced for more than one dependency in this Sensor object", dep.EventName, dep.EventSourceName)
			}
			comboKeys[comboKey] = true
		}

		if err := validateEventFilter(dep.Filters); err != nil {
			return err
		}

		if err := validateLogicalOperator(dep.FiltersLogicalOperator); err != nil {
			return err
		}
	}
	return nil
}

// validateLogicalOperator verifies that the logical operator in input is equal to a supported value
func validateLogicalOperator(logOp v1alpha1.LogicalOperator) error {
	if logOp != v1alpha1.AndLogicalOperator &&
		logOp != v1alpha1.OrLogicalOperator &&
		logOp != v1alpha1.EmptyLogicalOperator {
		return fmt.Errorf("logical operator %s not supported", logOp)
	}
	return nil
}

// validateComparator verifies that the comparator in input is equal to a supported value
func validateComparator(comp v1alpha1.Comparator) error {
	if comp != v1alpha1.GreaterThanOrEqualTo &&
		comp != v1alpha1.GreaterThan &&
		comp != v1alpha1.EqualTo &&
		comp != v1alpha1.NotEqualTo &&
		comp != v1alpha1.LessThan &&
		comp != v1alpha1.LessThanOrEqualTo &&
		comp != v1alpha1.EmptyComparator {
		return fmt.Errorf("comparator %s not supported", comp)
	}

	return nil
}

// validateEventFilter for a sensor
func validateEventFilter(filter *v1alpha1.EventDependencyFilter) error {
	if filter == nil {
		return nil
	}

	if err := validateLogicalOperator(filter.ExprLogicalOperator); err != nil {
		return err
	}

	if err := validateLogicalOperator(filter.DataLogicalOperator); err != nil {
		return err
	}

	if filter.Exprs != nil {
		for _, expr := range filter.Exprs {
			if err := validateEventExprFilter(&expr); err != nil {
				return err
			}
		}
	}

	if filter.Data != nil {
		for _, data := range filter.Data {
			if err := validateEventDataFilter(&data); err != nil {
				return err
			}
		}
	}

	if filter.Context != nil {
		if err := validateEventCtxFilter(filter.Context); err != nil {
			return err
		}
	}

	if filter.Time != nil {
		if err := validateEventTimeFilter(filter.Time); err != nil {
			return err
		}
	}
	return nil
}

// validateEventExprFilter validates context filter
func validateEventExprFilter(exprFilter *v1alpha1.ExprFilter) error {
	if exprFilter.Expr == "" ||
		len(exprFilter.Fields) == 0 {
		return fmt.Errorf("one of expr filters is not valid (expr and fields must be not empty)")
	}

	for _, fld := range exprFilter.Fields {
		if fld.Path == "" || fld.Name == "" {
			return fmt.Errorf("one of expr filters is not valid (path and name in a field must be not empty)")
		}
	}

	return nil
}

// validateEventDataFilter validates context filter
func validateEventDataFilter(dataFilter *v1alpha1.DataFilter) error {
	if dataFilter.Comparator != v1alpha1.EmptyComparator {
		if err := validateComparator(dataFilter.Comparator); err != nil {
			return err
		}
	}

	if dataFilter.Path == "" ||
		dataFilter.Type == "" ||
		len(dataFilter.Value) == 0 {
		return fmt.Errorf("one of data filters is not valid (type, path and value must be not empty)")
	}

	for _, val := range dataFilter.Value {
		if val == "" {
			return fmt.Errorf("one of data filters is not valid (value must be not empty)")
		}
	}

	return nil
}

// validateEventCtxFilter validates context filter
func validateEventCtxFilter(ctxFilter *v1alpha1.EventContext) error {
	if ctxFilter.Type == "" &&
		ctxFilter.Subject == "" &&
		ctxFilter.Source == "" &&
		ctxFilter.DataContentType == "" {
		return fmt.Errorf("no fields specified in ctx filter (aka all events will be discarded)")
	}
	return nil
}

// validateEventTimeFilter validates time filter
func validateEventTimeFilter(timeFilter *v1alpha1.TimeFilter) error {
	now := time.Now().UTC()

	// Parse start and stop
	startTime, startErr := common.ParseTime(timeFilter.Start, now)
	if startErr != nil {
		return startErr
	}
	stopTime, stopErr := common.ParseTime(timeFilter.Stop, now)
	if stopErr != nil {
		return stopErr
	}

	if stopTime.Equal(startTime) {
		return fmt.Errorf("invalid event time filter: stop '%s' is equal to start '%s", timeFilter.Stop, timeFilter.Start)
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
func validateK8sTriggerPolicy(policy *v1alpha1.K8SResourcePolicy) error {
	if policy == nil {
		return nil
	}
	if policy.Labels == nil {
		return fmt.Errorf("resource labels are not specified")
	}
	return nil
}

// validateStatusPolicy validates a http trigger policy
func validateStatusPolicy(policy *v1alpha1.StatusPolicy) error {
	if policy == nil {
		return nil
	}
	if policy.Allow == nil {
		return fmt.Errorf("list of allowed response status is not specified")
	}
	return nil
}
